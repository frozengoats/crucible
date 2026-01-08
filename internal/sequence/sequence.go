package sequence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/functions"
	"github.com/frozengoats/crucible/internal/log"
	"github.com/frozengoats/crucible/internal/render"
	"github.com/frozengoats/crucible/internal/ssh"
	"github.com/frozengoats/crucible/internal/utils"
	"github.com/frozengoats/eval"
	"github.com/frozengoats/kvstore"
	"github.com/goccy/go-yaml"
)

var nameValidator = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

const ImmediateKey string = "__immediate"

type ActionContext struct {
	Name        string
	Description string
	Context     json.RawMessage
	Error       string
}

type SeqPos struct {
	Name     string
	Context  *kvstore.Store
	Sequence *Sequence
	Position int
}

type Sync struct {
	Src           string `yaml:"src"`           // local resource(s) to sync to remote
	Dest          string `yaml:"dest"`          // remote location to sync to
	PreserveOwner bool   `yaml:"preserveOwner"` // preserve ownership
	PreservePerms bool   `yaml:"preservePerms"` // preserve file permissions
	PreserveGroup bool   `yaml:"preserveGroup"` // preserve group
}

type Until struct {
	PauseInterval float64 `yaml:"pauseInterval"` // interval in seconds to pause between next action execution if until condition is not met
	MaxAttempts   int     `yaml:"maxAttempts"`   // max attempts to execute the action if the condition is not met
	Condition     string  `yaml:"condition"`     // condition which must evaluate to true in order to stop execution
}

type Template struct {
	Src     string            `yaml:"src"`
	Dest    string            `yaml:"dest"`
	Context map[string]string `yaml:"context"`
}

type Import struct {
	Path    string            `yaml:"path"`
	Context map[string]string `yaml:"context"`
}

type Pause struct {
	Before float64 `yaml:"before"`
	After  float64 `yaml:"after"`
}

type Action struct {
	Name           string    `yaml:"name"`           // the name of the action, referrable from other actions (unnamed actions will not capture or retain data)
	Description    string    `yaml:"description"`    // action description
	Iterate        string    `yaml:"iterate"`        // if an iterable is provided, it will be iterated and the child action will be called for each element
	Import         *Import   `yaml:"import"`         // if specified, a sequence is imported from a location relative to the top level config.yaml
	When           string    `yaml:"when"`           // conditional expression which must evaluate to true, in order for the action or loop to be executed
	FailWhen       string    `yaml:"failWhen"`       // conditional expression which when evaluating to true indicates a failure (failures are otherwise implicit to command execution return codes)
	IgnoreExitCode bool      `yaml:"ignoreExitCode"` // ignores the exit code of an execution, so that it does not cause the sequence to terminate
	Until          *Until    `yaml:"until"`          // execute action until the condition evaluates to true
	Action         *Action   `yaml:"action"`         // action to be executed if an iterable is present as well
	ParseJson      bool      `yaml:"parseJson"`      // processes the standard output as JSON and makes the data available on the .kv context of the action
	ParseYaml      bool      `yaml:"parseYaml"`      // processes the standard output as YAML and makes the data available on the .kv context of the action
	Su             string    `yaml:"su"`             // switch to the following user (can be a name or base 10 string of a numeric id)
	Sudo           bool      `yaml:"sudo"`           // run the command as root
	SubSequence    *Sequence `yaml:"subSequence"`    // sub sequence if imported
	Local          bool      `yaml:"local"`          // when true, action will be executed locally instead of remotely, this is useful for preparing local assets which might need to be present locally but not remotely
	Pause          *Pause    `yaml:"pause"`          // pause for n seconds before and/or after the action

	// these properties are independent action properties, mutually exclusive
	Stdin    string    `yaml:"stdin"`    // only valid with exec/shell
	Exec     []string  `yaml:"exec"`     // execute a command
	Shell    string    `yaml:"shell"`    // execute a command using sh
	Sync     *Sync     `yaml:"sync"`     // sync files from local to remote
	Template *Template `yaml:"template"` // render a template
}

func (a *Action) Validate() error {
	if a.Name != "" {
		if !nameValidator.MatchString(a.Name) {
			return fmt.Errorf("action name \"%s\" is invalid, must contain only letter, numbers or underscores and cannot begin with a number", a.Name)
		}
	}

	return nil

}

type Sequence struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Sequence    []*Action `yaml:"sequence"`
	filename    string
}

func (s *Sequence) Validate() error {
	if s.Name != "" {
		if !nameValidator.MatchString(s.Name) {
			return fmt.Errorf("sequence name \"%s\" is invalid, must contain only letter, numbers or underscores and cannot begin with a number", s.Name)
		}
	}

	for _, a := range s.Sequence {
		err := a.Validate()
		if err != nil {
			return err
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

func LoadSequence(cwdPath string, filename string) (*Sequence, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to read sequence file %s\n%w", filename, err)
	}
	b = bytes.ReplaceAll(b, []byte("{{"), []byte("<!!"))
	b = bytes.ReplaceAll(b, []byte("}}"), []byte("!!>"))

	s := &Sequence{}
	err = yaml.Unmarshal(b, s)
	if err != nil {
		return nil, fmt.Errorf("sequence file %s contained bad yaml data\n%w", filename, err)
	}
	s.filename = filename
	if err := s.Validate(); err != nil {
		return nil, err
	}

	// iterate the actions in the sequence
	for _, a := range s.Sequence {
		if a.Import != nil {
			importPath, err := filepath.Abs(filepath.Join(cwdPath, a.Import.Path))
			if err != nil {
				return nil, fmt.Errorf("unable to resolve import path for %s\n%w", a.Import, err)
			}

			a.SubSequence, err = LoadSequence(cwdPath, importPath)
			if err != nil {
				return nil, fmt.Errorf("unable to load sub sequence at %s\n%w", importPath, err)
			}
			a.SubSequence.filename = importPath
			if err := a.SubSequence.Validate(); err != nil {
				return nil, err
			}

			a.SubSequence.Name = a.Name
		}
	}

	return s, err
}

type ExecutionInstance struct {
	config               *config.Config
	hostIdent            string
	hostConfig           *config.HostConfig
	executionClient      cmdsession.ExecutionClient
	localExecutionClient cmdsession.ExecutionClient
	sequence             *Sequence
	totalExecutionSteps  int
	executionStack       []SeqPos
	currentExecutionStep int
	ImmediateContexts    []*ActionContext
	ExecContext          *kvstore.Store // context accumulated through execution ()
	HostContext          *kvstore.Store // per host config context
	lock                 sync.Mutex

	err error
}

func (s *Sequence) NewExecutionInstance(executionClient cmdsession.ExecutionClient, config *config.Config, hostIdent string) (*ExecutionInstance, error) {
	var hostContext *kvstore.Store
	var err error
	hostContextSource := config.Hosts[hostIdent].Context
	if hostContextSource != nil {
		hostContext, err = kvstore.FromMapping(hostContextSource)
		if err != nil {
			return nil, fmt.Errorf("unable to set host context: %w", err)
		}
	} else {
		hostContext = kvstore.NewStore()
	}

	return &ExecutionInstance{
		config:               config,
		hostIdent:            hostIdent,
		hostConfig:           config.Hosts[hostIdent],
		executionClient:      executionClient,
		localExecutionClient: cmdsession.NewLocalExecutionClient(),
		sequence:             s,
		totalExecutionSteps:  s.CountExecutionSteps(),
		HostContext:          hostContext,
	}, nil
}

func (ei *ExecutionInstance) SetError(err error) {
	ei.err = err
}

func (ei *ExecutionInstance) GetCurrentNamespace() []any {
	var curNs []any
	for _, s := range ei.executionStack {
		curNs = append(curNs, s.Sequence.Name)
	}

	return curNs
}

func (ei *ExecutionInstance) HasMore() bool {
	ei.lock.Lock()
	defer ei.lock.Unlock()

	if ei.err != nil {
		return false
	}

	return ei.currentExecutionStep < ei.totalExecutionSteps
}

func (ei *ExecutionInstance) GetError() error {
	ei.lock.Lock()
	defer ei.lock.Unlock()

	return ei.err
}

// Next returns the next unexecuted action in the sequence, or nil if none remain
func (ei *ExecutionInstance) Next() (*Action, error) {
	ei.lock.Lock()
	defer ei.lock.Unlock()

	if ei.err != nil {
		return nil, nil
	}

	var stackIndex int
	var stackItem *SeqPos
	started := false

	if ei.executionStack == nil {
		started = true
		ei.executionStack = []SeqPos{
			{
				Context:  kvstore.NewStore(),
				Sequence: ei.sequence,
				Position: 0,
			},
		}
	}

	for {
		if len(ei.executionStack) == 0 {
			return nil, nil
		}

		stackIndex = len(ei.executionStack) - 1
		stackItem = &ei.executionStack[stackIndex]

		if !started {
			stackItem.Position++
		}

		if stackItem.Position >= len(stackItem.Sequence.Sequence) {
			// pop this item off the stack and move onto the next
			// when the stack is collapsed, the last context is written to the parent context, under
			// the key representing the name of the collapsed sequence
			if len(ei.executionStack) > 1 {
				lastExecutionItem := ei.executionStack[len(ei.executionStack)-1]
				ei.executionStack = ei.executionStack[:len(ei.executionStack)-1]
				currentExecutionItem := ei.executionStack[len(ei.executionStack)-1]
				if lastExecutionItem.Name != "" {
					err := currentExecutionItem.Context.Set(lastExecutionItem.Context.GetMapping(), lastExecutionItem.Name)
					if err != nil {
						return nil, err
					}
				}
			} else {
				ei.executionStack = []SeqPos{}
			}
		} else {
			action := stackItem.Sequence.Sequence[stackItem.Position]
			if action.SubSequence == nil {
				ei.currentExecutionStep++
				ei.ExecContext = ei.executionStack[len(ei.executionStack)-1].Context
				return action, nil
			}

			isSatisfied, err := ei.whenSatisfied(action)
			if !isSatisfied {
				context := []any{
					"host", ei.hostIdent,
				}
				log.Info(context, "skipping due to falsey when clause")
				continue
			}

			// push the next subsequence onto the stack, seed any context with the context from the import step
			var newContext *kvstore.Store
			if action.Import != nil && action.Import.Context != nil {
				evalContext := map[string]any{}
				for k, v := range action.Import.Context {
					evalV, err := render.Render(v, ei.variableLookup, functions.Call)
					if err != nil {
						return nil, fmt.Errorf("problem evaluating sequence context value \"%s\" at key \"%s\": %w", v, k, err)
					}

					evalContext[k] = evalV
				}

				newContext, err = kvstore.FromMapping(evalContext)
				if err != nil {
					return nil, fmt.Errorf("unable to construct sequence context: %w", err)
				}
			} else {
				newContext = kvstore.NewStore()
			}

			ei.executionStack = append(ei.executionStack, SeqPos{
				Name:     action.Name,
				Context:  newContext,
				Sequence: action.SubSequence,
				Position: -1,
			})
		}
	}
}

func (ei *ExecutionInstance) variableLookup(key string) (any, error) {
	var store *kvstore.Store
	if strings.HasPrefix(key, ".Values.") {
		key = strings.TrimPrefix(key, ".Values.")
		store = ei.config.ValuesStore
	} else if strings.HasPrefix(key, ".Context.") {
		key = strings.TrimPrefix(key, ".Context.")
		store = ei.ExecContext
	} else if strings.HasPrefix(key, ".Host.") {
		key = strings.TrimPrefix(key, ".Host.")
		store = ei.HostContext
	} else {
		// assume immediate context if not prefixed by one of the two known namespace classifiers
		key = fmt.Sprintf("%s%s", ImmediateKey, key)
		store = ei.ExecContext
	}

	return store.Get(kvstore.ParseNamespaceString(key)...), nil
}

func (ei *ExecutionInstance) Close() error {
	return ei.executionClient.Close()
}

func (ei *ExecutionInstance) whenSatisfied(action *Action) (bool, error) {
	if action.When == "" {
		return true, nil
	}

	// evaluate the when condition
	whenResult, err := eval.Evaluate(action.When, ei.variableLookup, functions.Call)
	if err != nil {
		return false, fmt.Errorf("unable to evaluate when clause: %s\n%w", action.When, err)
	}

	return eval.IsTruthy(whenResult), nil
}

func (ei *ExecutionInstance) Execute(action *Action) error {
	context := []any{
		"host", ei.hostIdent,
	}
	log.Info(context, "processing action \"%s\"", action.Description)

	if action.Pause != nil && action.Pause.Before > 0 {
		log.Debug(context, "pausing before action execution for %0.2f seconds", action.Pause.Before)
		time.Sleep(time.Second * time.Duration(action.Pause.Before))
	}

	isSatisfied, err := ei.whenSatisfied(action)
	if err != nil {
		return err
	}

	if !isSatisfied {
		log.Info(context, "skipping due to falsey when clause")
		return nil
	}

	if action.Iterate != "" {
		iterableResult, err := eval.Evaluate(action.Iterate, ei.variableLookup, functions.Call)
		if err != nil {
			return fmt.Errorf("unable to evaluate interable attribute: %s\n%w", action.Iterate, err)
		}

		iterableArray, ok := iterableResult.([]any)
		if !ok {
			return fmt.Errorf("iterate attribute does not return an array")
		}

		for i, item := range iterableArray {
			err = ei.ExecContext.Set(item, ImmediateKey, "item")
			if err != nil {
				return err
			}
			action.Action.Description = fmt.Sprintf("%s (iteration %d of %d)", action.Description, i+1, len(iterableArray))
			err = ei.Execute(action.Action)
			if err != nil {
				return err
			}
		}

		// since iterables call an internal action, once this is done, there's no continuing
		return nil
	}

	// this for loop will break immediately unless an until clause is set
	var stdout []byte
	var exitCode int
	untilAttempts := 0
	for {
		stdout, exitCode, err = ei.executeSingleAction(action)
		if err != nil {
			ei.ImmediateContexts = append(ei.ImmediateContexts, &ActionContext{
				Name:        action.Name,
				Description: action.Description,
				Error:       err.Error(),
			})
			return err
		}

		log.Debug(context, "exit code: %d", exitCode)
		log.Debug(context, "stdout\n%s", string(stdout))
		err = ei.ExecContext.Set(string(stdout), ImmediateKey, "stdout")
		if err != nil {
			return err
		}
		err = ei.ExecContext.Set(exitCode, ImmediateKey, "exitCode")
		if err != nil {
			return err
		}

		if ei.config.Debug {
			jBytes, err := json.Marshal(ei.ExecContext.GetMapping(ImmediateKey))
			if err != nil {
				return fmt.Errorf("unable to export immediate context: %w", err)
			}
			ei.ImmediateContexts = append(ei.ImmediateContexts, &ActionContext{
				Name:        action.Name,
				Description: action.Description,
				Context:     jBytes,
			})
		}

		if exitCode == 0 {
			if action.ParseJson {
				jsonMap := map[string]any{}
				err = json.Unmarshal(stdout, &jsonMap)
				if err != nil {
					return fmt.Errorf("unable to unmarshal json from stdout: %w", err)
				}
				err = ei.ExecContext.Set(jsonMap, ImmediateKey, "json")
				if err != nil {
					return err
				}
			}

			if action.ParseYaml {
				yamlMap := map[string]any{}
				err = yaml.Unmarshal(stdout, &yamlMap)
				if err != nil {
					return fmt.Errorf("unable to unmarshal yaml from stdout: %w", err)
				}
				err = ei.ExecContext.Set(yamlMap, ImmediateKey, "yaml")
				if err != nil {
					return err
				}
			}
		}

		if action.Until == nil {
			break
		}

		untilResult, err := eval.Evaluate(action.Until.Condition, ei.variableLookup, functions.Call)
		if err != nil {
			return fmt.Errorf("unable to evaluate until condition: %w", err)
		}
		if eval.IsTruthy(untilResult) {
			break
		}

		untilAttempts++
		if untilAttempts >= action.Until.MaxAttempts {
			return fmt.Errorf("maximum number of attempts occurred and until clause requirement was not met")
		}
	}

	if exitCode != 0 {
		if !action.IgnoreExitCode {
			return cmdsession.NewExitCodeError(exitCode)
		}
	}

	if action.FailWhen != "" {
		failWhenResult, err := eval.Evaluate(action.FailWhen, ei.variableLookup, functions.Call)
		if err != nil {
			return fmt.Errorf("unable to evaluate failWhen condition %s: %w", action.FailWhen, err)
		}

		if eval.IsTruthy(failWhenResult) {
			return fmt.Errorf("condition of failWhen clause evaluated to true")
		}
	}

	// at this point it is safe to propagate all transient data to the context, if context names exist
	if action.Name != "" {
		actionLocalNamespace := action.Name
		err = ei.ExecContext.Set(ei.ExecContext.GetMapping(ImmediateKey), actionLocalNamespace)
		if err != nil {
			return fmt.Errorf("unable to set fully local context data on store: %w", err)
		}
	}

	if action.Pause != nil && action.Pause.After > 0 {
		log.Debug(context, "pausing after action execution for %0.2f seconds", action.Pause.After)
		time.Sleep(time.Second * time.Duration(action.Pause.After))
	}

	return nil
}

func (ei *ExecutionInstance) getSuUser(action *Action) (string, error) {
	result, err := render.Render(action.Su, ei.variableLookup, functions.Call)
	if err != nil {
		return "", nil
	}
	return render.ToString(result), nil
}

func (ei *ExecutionInstance) getExecString(action *Action) ([]string, error) {
	var renderedExec []string
	for _, ex := range action.Exec {
		rendEx, err := render.Render(ex, ei.variableLookup, functions.Call)
		if err != nil {
			return nil, fmt.Errorf("unable to template action exec command portion: %w", err)
		}
		renderedExec = append(renderedExec, render.ToString(rendEx))
	}

	if !action.Sudo && action.Su == "" {
		return renderedExec, nil
	}

	if action.Sudo {
		renderedExec = append([]string{"sudo"}, renderedExec...)
	} else if action.Su != "" {
		suUser, err := ei.getSuUser(action)
		if err != nil {
			return nil, err
		}
		renderedExec = append([]string{"sudo", "-H", "-u", suUser}, renderedExec...)
	}

	return renderedExec, nil
}

func (ei *ExecutionInstance) getShellString(action *Action) ([]string, error) {
	var renderedExec []string
	rendEx, err := render.Render(action.Shell, ei.variableLookup, functions.Call)
	if err != nil {
		return nil, fmt.Errorf("unable to template action shell command portion: %w", err)
	}

	combined := utils.Combine(render.ToString(rendEx))

	if !action.Sudo && action.Su == "" {
		return []string{ei.config.Executor.ShellBinary, "-c", combined}, nil
	}
	if action.Sudo {
		renderedExec = []string{"sudo", ei.config.Executor.ShellBinary, "-c", combined}
	} else if action.Su != "" {
		suUser, err := ei.getSuUser(action)
		if err != nil {
			return nil, err
		}
		renderedExec = []string{"sudo", "-H", "-u", suUser, ei.config.Executor.ShellBinary, "-c", combined}
	}

	return renderedExec, nil
}

// executeSingleAction performs the action execution, returning the output, exit code, and or any error
func (ei *ExecutionInstance) executeSingleAction(action *Action) ([]byte, int, error) {
	var err error
	if action.Shell != "" && len(action.Exec) > 0 {
		return nil, 0, fmt.Errorf("shell and exec directives are mutually exclusive")
	}

	if action.Shell != "" || len(action.Exec) > 0 {
		var execStr []string
		if action.Shell != "" {
			execStr, err = ei.getShellString(action)
			if err != nil {
				return nil, 0, err
			}
		} else if len(action.Exec) > 0 {
			execStr, err = ei.getExecString(action)
			if err != nil {
				return nil, 0, err
			}
		}

		var reader io.Reader
		if action.Stdin != "" {
			stdin, err := render.Render(action.Stdin, ei.variableLookup, functions.Call)
			if err != nil {
				return nil, 0, fmt.Errorf("unable to evaluate action stdin")
			}

			switch t := stdin.(type) {
			case []byte:
				reader = bytes.NewReader(t)
			case string:
				reader = strings.NewReader(t)
			default:
				return nil, 0, fmt.Errorf("action stdin must evaluate to a string or byte array (it is currently %T)", t)
			}
		}
		var execClient cmdsession.ExecutionClient
		if action.Local {
			execClient = ei.localExecutionClient
		} else {
			execClient = ei.executionClient
		}
		return ei.executeRemoteCommand(execClient, reader, execStr)
	}

	// if the code gets to this point, it's a sync
	if action.Sync != nil {
		return nil, 0, ei.sync(action)
	}

	if action.Template != nil {
		return ei.template(action)
	}

	return nil, 0, nil
}

func (ei *ExecutionInstance) executeRemoteCommand(execClient cmdsession.ExecutionClient, stdin io.Reader, cmd []string) ([]byte, int, error) {
	// create a new command session
	var output []byte
	attempts := 0
	for {
		sess, err := execClient.NewCmdSession()
		if err != nil {
			return nil, 0, err
		}

		output, err = sess.Execute(stdin, cmd...)
		if err != nil {
			_, ok := err.(*cmdsession.SessionError)
			if ok {
				log.Debug(nil, "waiting %0.2f seconds before attempting SSH retry after failure", ei.config.Executor.Ssh.DelayAfterConnectionFailure)
				_ = execClient.Close()
				for {
					err = execClient.Connect()
					if err != nil {
						log.Debug(nil, "%s", err.Error())
						attempts++
						if attempts >= ei.config.Executor.Ssh.MaxConnectionAttempts {
							return nil, 0, err
						}

						time.Sleep(time.Duration(ei.config.Executor.Ssh.DelayAfterConnectionFailure) * time.Second)
						continue
					}
					break
				}
			}

			exitCode, hasExitCode := cmdsession.GetExitCode(err)
			if !hasExitCode {
				return nil, 0, err
			}

			return output, exitCode, nil
		}

		return output, 0, nil
	}
}

func (ei *ExecutionInstance) sync(action *Action) error {
	syncAction := action.Sync

	return ssh.Rsync(ei.config.Username(ei.hostIdent), ei.config.Hostname(ei.hostIdent), ei.config.Port(ei.hostIdent), ei.config.KeyPath(ei.hostIdent), syncAction.Src, syncAction.Dest)
}

// template causes the templatization of a local resource and renders it to a remote location
func (ei *ExecutionInstance) template(action *Action) ([]byte, int, error) {
	var err error
	templateAction := action.Template

	src := templateAction.Src
	srcAny, err := render.Render(src, ei.variableLookup, functions.Call)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to evaluate template src: %w", err)
	}
	src = render.ToString(srcAny)

	dest := templateAction.Dest
	destAny, err := render.Render(dest, ei.variableLookup, functions.Call)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to evaluate template dest: %w", err)
	}
	dest = render.ToString(destAny)

	if !filepath.IsAbs(src) {
		p, err := filepath.Abs(filepath.Join(ei.config.CwdPath, src))
		if err != nil {
			return nil, 0, fmt.Errorf("unable to transform template path %s to abs path: %w", src, err)
		}

		src = p
	}
	t, err := template.ParseFiles(src)
	if err != nil {
		return nil, 0, err
	}

	var rendered bytes.Buffer
	evalContext := map[string]any{}
	for k, v := range templateAction.Context {
		evalV, err := render.Render(v, ei.variableLookup, functions.Call)
		if err != nil {
			return nil, 0, fmt.Errorf("problem evaluating value \"%s\" at key \"%s\"", v, k)
		}

		evalContext[k] = evalV
	}
	err = t.Execute(&rendered, evalContext)
	if err != nil {
		return nil, 0, err
	}

	var execStr []string
	shellStr := utils.Combine(fmt.Sprintf("cat > %s", dest))
	if action.Sudo {
		execStr = []string{"sudo", ei.config.Executor.ShellBinary, "-c", shellStr}
	} else if action.Su != "" {
		suUser, err := ei.getSuUser(action)
		if err != nil {
			return nil, 0, err
		}
		execStr = []string{"sudo", "-H", "-u", suUser, ei.config.Executor.ShellBinary, "-c", shellStr}
	} else {
		execStr = []string{ei.config.Executor.ShellBinary, "-c", shellStr}
	}

	return ei.executeRemoteCommand(ei.executionClient, &rendered, execStr)
}
