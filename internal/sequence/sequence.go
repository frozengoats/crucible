package sequence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/goccy/go-yaml"
)

var EndOfSequence = errors.New("end of sequence reached")

type SeqPos struct {
	Sequence *Sequence
	Position int
}

type Sync struct {
	Local     string `yaml:"local"`     // local resource(s) to sync to remote
	Remote    string `yaml:"remote"`    // remote location to sync to
	FilePerms string `yaml:"filePerms"` // permissions to apply to all files
	DirPerms  string `yaml:"dirPerms"`  // permissions to apply to all directories
	Owner     string `yaml:"owner"`     // ownership to apply to all files and directories
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

type Sequence struct {
	Description string
	Sequence    []*Action
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
		}
	}

	return s, err
}

type ExecutionInstance struct {
	config               *config.Config
	hostIdent            string
	hostConfig           *config.HostConfig
	executionClient      *cmdsession.ExecutionClient
	sequence             *Sequence
	totalExecutionSteps  int
	executionStack       []SeqPos
	currentExecutionStep int
}

func (s *Sequence) NewExecutionInstance(executionClient cmdsession.ExecutionClient, config *config.Config, hostIdent string) *ExecutionInstance {
	return &ExecutionInstance{
		config:              config,
		hostIdent:           hostIdent,
		hostConfig:          config.Hosts[hostIdent],
		executionClient:     &executionClient,
		sequence:            s,
		totalExecutionSteps: s.CountExecutionSteps(),
	}
}

func (ei *ExecutionInstance) HasMore() bool {
	return ei.currentExecutionStep < ei.totalExecutionSteps
}

func (ei *ExecutionInstance) Next() *Action {
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
	return nil
}
