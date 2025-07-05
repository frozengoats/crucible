package sequence

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/kvstore"
	"github.com/goccy/go-yaml"
)

var EndOfSequence = errors.New("end of sequence reached")

type SeqPos struct {
	Sequence *Sequence
	Position int
}

type Sync struct {
	Local         string `yaml:"local"`         // local resource(s) to sync to remote
	Remote        string `yaml:"remote"`        // remote location to sync to
	PreserveOwner bool   `yaml:"preserveOwner"` // preserve ownership
	PreservePerms bool   `yaml:"preservePerms"` // preserve file permissions
	PreserveGroup bool   `yaml:"preserveGroup"` // preserve group
}

type Until struct {
	PauseInterval float64 `yaml:"pauseInterval"` // interval in seconds to pause between next action execution if until condition is not met
	MaxAttempts   int     `yaml:"maxAttempts"`   // max attempts to execute the action if the condition is not met
	Condition     string  `yaml:"condition"`     // condition which must evaluate to true in order to stop execution
}

type Action struct {
	Name                string    `yaml:"name"` // the name of the action, referrable from other actions (unnamed actions will not capture or retain data)
	Description         string    `yaml:"description"`
	Iterable            string    `yaml:"iterable"`            // if an iterable is provided, it will be iterated and the child action will be called for each element
	Import              string    `yaml:"import"`              // if specified, the action/sequence is imported from a location relative to the top level config.yaml
	When                string    `yaml:"when"`                // conditional expression which must evaluate to true, in order for the action or loop to be executed
	FailWhen            string    `yaml:"failWhen"`            // conditional expression which when evaluating to true indicates a failure (failures are otherwise implicit to command execution return codes)
	StoreSuccessAsTrue  bool      `yaml:"storeSuccessAsTrue"`  // stores a success as true and a failure as false, implies that a failure does not cancel execution of the next action
	StoreSuccessAsFalse bool      `yaml:"storeSuccessAsFalse"` // stores a success as false and a failure as true, implies that a failure does not cancel execution of the next action
	Until               *Until    `yaml:"until"`               // execute action until the condition evaluates to true
	Action              *Action   `yaml:"action"`              // action to be executed if an iterable is present as well
	ParseJson           bool      `yaml:"parseJson"`           // processes the standard output as JSON and makes the data available on the .kv context of the action
	ParseYaml           bool      `yaml:"parseYaml"`           // processes the standard output as YAML and makes the data available on the .kv context of the action
	Su                  string    `yaml:"su"`                  // switch to the following user (can be a name or base 10 string of a numeric id)
	Sudo                bool      `yaml:"sudo"`                // run the command as root
	SubSequence         *Sequence `yaml:"subSequence"`         // sub sequence if imported

	// these properties are independent action properties, mutually exclusive
	Exec  []string `yaml:"exec"`  // execute a command
	Shell string   `yaml:"shell"` // execute a command using sh
	Sync  *Sync    `yaml:"sync"`  // sync files from local to remote
}

func (a *Action) GetExecutionString() (string, bool) {
	if len(a.Exec) == 0 || len(a.Shell) == 0 {
		return "", false
	}

	if len(a.Exec) > 0 {
		return strings.Join(a.Exec, " "), true
	}

	return fmt.Sprintf("sh -c '%s'", a.Shell), true
}

type Sequence struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Sequence    []*Action `yaml:"actions"`
	filename    string
}

func (s *Sequence) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("sequence at %s must have the name field set", s.filename)
	}

	for index, action := range s.Sequence {
		if action.Name == "" {
			return fmt.Errorf("action at index %d in sequence file %s must have the name field set", index, s.filename)
		}
	}

	return nil
}

func (s *Sequence) CountExecutionSteps() int {
	steps := 0
	for _, seq := range s.Sequence {
		if seq.SubSequence != nil {
			steps += seq.SubSequence.CountExecutionSteps()
		} else {
			steps++
		}
	}

	return steps
}

func LoadSequence(filename string) (*Sequence, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to read sequence file %s\n%w", filename, err)
	}

	s := &Sequence{}
	err = yaml.Unmarshal(b, s)
	if err != nil {
		return nil, fmt.Errorf("sequence file %s contained bad yaml data\n%w", filename, err)
	}
	s.filename = filename
	if err := s.Validate(); err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("unable to obtain the current working directory\n%w", err)
	}

	for _, s := range s.Sequence {
		if s.Import != "" {
			importPath, err := filepath.Abs(filepath.Join(cwd, s.Import))
			if err != nil {
				return nil, fmt.Errorf("unable to resolve import path for %s\n%w", s.Import, err)
			}

			s.SubSequence, err = LoadSequence(importPath)
			if err != nil {
				return nil, fmt.Errorf("unable to load sub sequence at %s\n%w", importPath, err)
			}
			s.SubSequence.filename = importPath
			if err := s.SubSequence.Validate(); err != nil {
				return nil, err
			}
		}
	}

	return s, err
}

type ExecutionInstance struct {
	config               *config.Config
	hostIdent            string
	hostConfig           *config.HostConfig
	executionClient      cmdsession.ExecutionClient
	sequence             *Sequence
	totalExecutionSteps  int
	executionStack       []SeqPos
	currentExecutionStep int

	varContext  *kvstore.Store // variable context
	execContext *kvstore.Store // context accumulated through execution ()
	lock        sync.Mutex
}

func (s *Sequence) NewExecutionInstance(executionClient cmdsession.ExecutionClient, config *config.Config, hostIdent string) *ExecutionInstance {
	return &ExecutionInstance{
		config:              config,
		hostIdent:           hostIdent,
		hostConfig:          config.Hosts[hostIdent],
		executionClient:     executionClient,
		sequence:            s,
		totalExecutionSteps: s.CountExecutionSteps(),
		varContext:          kvstore.NewStore(),
		execContext:         kvstore.NewStore(),
	}
}

func (ei *ExecutionInstance) GetCurrentNamespace() []string {
	var curNs []string
	for _, s := range ei.executionStack {
		curNs = append(curNs, s.Sequence.Name)
	}

	return curNs
}

func (ei *ExecutionInstance) HasMore() bool {
	ei.lock.Lock()
	defer ei.lock.Unlock()

	return ei.currentExecutionStep < ei.totalExecutionSteps
}

// Next returns the next unexecuted action in the sequence, or nil if none remain
func (ei *ExecutionInstance) Next() *Action {
	ei.lock.Lock()
	defer ei.lock.Unlock()

	var stackIndex int
	var stackItem *SeqPos
	started := false

	if ei.executionStack == nil {
		started = true
		ei.executionStack = []SeqPos{
			{
				Sequence: ei.sequence,
				Position: 0,
			},
		}
	}

	for {
		if len(ei.executionStack) == 0 {
			return nil
		}

		stackIndex = len(ei.executionStack) - 1
		stackItem = &ei.executionStack[stackIndex]

		if !started {
			stackItem.Position++
		}

		if stackItem.Position >= len(stackItem.Sequence.Sequence) {
			// pop this item off the stack and move onto the next
			if len(ei.executionStack) > 1 {
				ei.executionStack = ei.executionStack[:len(ei.executionStack)-1]
			} else {
				ei.executionStack = []SeqPos{}
			}
		} else {
			action := stackItem.Sequence.Sequence[stackItem.Position]
			if action.SubSequence == nil {
				ei.currentExecutionStep++
				return action
			}

			// push the next subsequence onto the stack
			ei.executionStack = append(ei.executionStack, SeqPos{
				Sequence: action.SubSequence,
				Position: -1,
			})
		}
	}
}

func (ei *ExecutionInstance) Execute(action *Action) error {
	parentNamespace := ei.GetCurrentNamespace()
	var actionFqNamespace []string
	actionFqNamespace = append(actionFqNamespace, parentNamespace...)
	actionFqNamespace = append(actionFqNamespace, action.Name)

	actionNamespace := action.Name

	// if action.Iterable {

	// }

	// Name           string    `yaml:"name"` // the name of the action, referrable from other actions (unnamed actions will not capture or retain data)
	// Description         string    `yaml:"description"`
	// Iterable            string    `yaml:"iterable"`            // if an iterable is provided, it will be iterated and the child action will be called for each element
	// Import              string    `yaml:"import"`              // if specified, the action/sequence is imported from a location relative to the top level config.yaml
	// When                string    `yaml:"when"`                // conditional expression which must evaluate to true, in order for the action or loop to be executed
	// FailWhen            string    `yaml:"failWhen"`            // conditional expression which when evaluating to true indicates a failure (failures are otherwise implicit to command execution return codes)
	// StoreSuccessAsTrue  bool      `yaml:"storeSuccessAsTrue"`  // stores a success as true and a failure as false, implies that a failure does not cancel execution of the next action
	// StoreSuccessAsFalse bool      `yaml:"storeSuccessAsFalse"` // stores a success as false and a failure as true, implies that a failure does not cancel execution of the next action
	// Until               *Until    `yaml:"until"`               // execute action until the condition evaluates to true
	// Action              *Action   `yaml:"action"`              // action to be executed if an iterable is present as well
	// ParseJson           bool      `yaml:"parseJson"`           // processes the standard output as JSON and makes the data available on the .kv context of the action
	// ParseYaml           bool      `yaml:"parseYaml"`           // processes the standard output as YAML and makes the data available on the .kv context of the action
	// Su                  string    `yaml:"su"`                  // switch to the following user (can be a name or base 10 string of a numeric id)
	// Sudo                bool      `yaml:"sudo"`                // run the command as root
	// SubSequence         *Sequence `yaml:"subSequence"`         // sub sequence if imported

	// these properties are independent action properties, mutually exclusive
	// Exec  []string `yaml:"exec"`  // execute a command
	// Shell string   `yaml:"shell"` // execute a command using sh
	// Sync  *Sync    `yaml:"sync"`  // sync files from local to remote

	return nil
}

// executeSingleAction performs the action execution, returning the output, exit code, and or any error
func (ei *ExecutionInstance) executeSingleAction(action *Action) ([]byte, int, error) {
	execStr, ok := action.GetExecutionString()
	if ok {
		return ei.executeRemoteCommand(execStr)
	}

	// if the code gets to this point, it's a sync
	if action.Sync != nil {
		return nil, 0, ei.sync(action)
	}

	return nil, 0, fmt.Errorf("unknown action execution")
}

func (ei *ExecutionInstance) executeRemoteCommand(commandStr string) ([]byte, int, error) {
	// create a new command session
	sess, err := ei.executionClient.NewCmdSession()
	if err != nil {
		return nil, 0, err
	}

	output, err := sess.Execute(commandStr)
	if err != nil {
		exitCode, hasExitCode := cmdsession.GetExitCode(err)
		if !hasExitCode {
			return nil, 0, err
		}

		return output, exitCode, nil
	}

	return output, 0, nil
}

func (ei *ExecutionInstance) sync(action *Action) error {
	syncAction := action.Sync
	cmd := exec.Command("rsync", syncAction.Local, fmt.Sprintf("%s@%s:%s", ei.config.User.Username, ei.hostConfig.Host, syncAction.Remote))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
