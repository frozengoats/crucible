package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/executor"
)

var command struct {
	Cwd         string   `short:"c" help:"change the current working directory to this location"`
	ConfigStack []string `short:"s" help:"list of paths to any config yaml overrides, stackable in order of occurrence"`
	Sequence    string   `arg:"" help:"the full or relative path to the sequence to execute"`
	Targets     []string `arg:"" help:"named machine targets and/or groups against which to execute the sequence"`
}

func runExecutor(execChan <-chan *executor.Executor, errChan chan error, syncExecutionSteps bool, wg *sync.WaitGroup) {
	for e := range execChan {
		if syncExecutionSteps {
			err := e.RunOne()
			if err != nil {
				errChan <- err
			}
		} else {
			err := e.RunAll()
			if err != nil {
				errChan <- err
			}
		}

		wg.Done()
	}
}

func run() error {
	var (
		cwd string
		err error
	)

	if command.Cwd != "" {
		cwd = command.Cwd
	} else {
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	configPath := filepath.Join(cwd, "config.yaml")

	_, err = os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("no config.yaml could be located at %s, are you sure this is a crucible configuration?\n%w", configPath, err)
	}

	configPaths := append([]string{configPath}, command.ConfigStack...)
	configObj, err := config.FromFilePaths(cwd, configPaths...)
	if err != nil {
		return err
	}

	selectedHosts := map[string]struct{}{}
	for _, hostIdent := range command.Targets {
		selectedHosts[hostIdent] = struct{}{}
	}

	hostIdents := []string{}
	for hostIdent, hostConfig := range configObj.Hosts {
		if len(selectedHosts) == 0 {
			hostIdents = append(hostIdents, hostIdent)
		} else {
			if _, ok := selectedHosts[hostIdent]; ok {
				hostIdents = append(hostIdents, hostIdent)
			} else if _, ok := selectedHosts[hostConfig.Group]; ok {
				hostIdents = append(hostIdents, hostIdent)
			}
		}
	}

	if len(hostIdents) == 0 {
		return fmt.Errorf("no hosts specified")
	}

	maxConcurrentHosts := configObj.Executor.MaxConcurrentHosts
	if len(hostIdents) < maxConcurrentHosts {
		maxConcurrentHosts = len(hostIdents)
	}

	// iterate the selected hosts
	executors := []*executor.Executor{}
	for _, hostIdent := range hostIdents {
		e, err := executor.NewExecutor(configObj, hostIdent, command.Sequence)
		if err != nil {
			return fmt.Errorf("unable to create executor\n%w", err)
		}
		executors = append(executors, e)
	}

	execWaitGroup := &sync.WaitGroup{}
	execChan := make(chan *executor.Executor, maxConcurrentHosts)
	errChan := make(chan error, len(hostIdents))
	for range maxConcurrentHosts {
		go runExecutor(execChan, errChan, configObj.Executor.SyncExecutionSteps, execWaitGroup)
	}

	if configObj.Executor.SyncExecutionSteps {
		hasMore := true
		for hasMore {
			hasMore = false
			for _, e := range executors {
				if e.HasMore() {
					hasMore = true
					execWaitGroup.Add(1)
					execChan <- e
				}
			}

			// wait until all goroutines are finished
			execWaitGroup.Wait()

			// pull errors from error channel
			errChanLen := len(errChan)
			for range errChanLen {
				err := <-errChan
				fmt.Printf("%s\n", err)
			}
			if errChanLen > 0 {
				close(execChan)
				return fmt.Errorf("sequence aborted due to one or more failures")
			}
		}
		close(execChan)
	} else {
		for _, e := range executors {
			execWaitGroup.Add(1)
			execChan <- e
		}

		// wait until all goroutines are finished
		execWaitGroup.Wait()

		// pull errors from error channel
		errChanLen := len(errChan)
		for range errChanLen {
			err := <-errChan
			fmt.Printf("%s\n", err)
		}
		if errChanLen > 0 {
			close(execChan)
			return fmt.Errorf("sequence aborted due to one or more failures")
		}
	}

	return nil
}

func main() {
	_ = kong.Parse(&command)
	err := run()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
