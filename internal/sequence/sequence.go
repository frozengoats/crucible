package sequence

import (
	"fmt"
	"os"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/goccy/go-yaml"
)

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
	// these properties are action metadata

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

	return s, err
}

type ExecutionInstance struct {
	syncEachStep    bool
	executionClient cmdsession.ExecutionClient
	seq             []*Action
	curStep         int
}

func (s *Sequence) CreateExecutionInstance(executionClient cmdsession.ExecutionClient, syncEachStep bool) *ExecutionInstance {
	return &ExecutionInstance{
		syncEachStep:    syncEachStep,
		executionClient: executionClient,
		seq:             s.Sequence[:],
	}
}

func (ei *ExecutionInstance) Execute() error {
	return nil
}
