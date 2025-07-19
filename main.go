package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/executor"
	"github.com/frozengoats/crucible/internal/log"
	"github.com/frozengoats/kvstore"
	"github.com/goccy/go-yaml"
)

var Version string = "dev"

var command struct {
	Cwd      string   `short:"c" help:"change the current working directory to this location"`
	Configs  []string `short:"s" help:"list of paths to any config yaml overrides, stackable in order of occurrence (excluding config.yaml)"`
	Values   []string `short:"v" help:"list of paths to values files, stackable in order of occurrence (excluding values.yaml)"`
	Sequence string   `arg:"" help:"the full or relative path to the sequence to execute"`
	Targets  []string `arg:"" help:"named machine targets and/or groups against which to execute the sequence"`
	Debug    bool     `short:"d" help:"enable debug mode"`
	Version  bool     `help:"display the current version"`
	Json     bool     `short:"j" help:"output results in json format, suppress normal logging"`
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

	mainConfigPath := filepath.Join(cwd, "config.yaml")
	_, err = os.Stat(mainConfigPath)
	if err != nil {
		return fmt.Errorf("no config.yaml could be located at %s, are you sure this is a crucible configuration?\n%w", mainConfigPath, err)
	}

	for i, configPath := range command.Configs {
		if !filepath.IsAbs(configPath) {
			absPath, err := filepath.Abs(filepath.Join(cwd, configPath))
			if err != nil {
				return fmt.Errorf("problem interpreting path %s\n%w", configPath, err)
			}
			command.Configs[i] = absPath
		}
	}
	configPaths := append([]string{mainConfigPath}, command.Configs...)

	mainValuesPath := filepath.Join(cwd, "values.yaml")
	for i, valuesPath := range command.Values {
		if !filepath.IsAbs(valuesPath) {
			absPath, err := filepath.Abs(filepath.Join(cwd, valuesPath))
			if err != nil {
				return fmt.Errorf("problem interpreting path %s\n%w", valuesPath, err)
			}
			command.Values[i] = absPath
		}
	}

	var valuesPaths []string
	_, err = os.Stat(mainConfigPath)
	if err == nil {
		valuesPaths = append([]string{mainValuesPath}, command.Values...)
	} else {
		valuesPaths = command.Values
	}

	if !filepath.IsAbs(command.Sequence) {
		absPath, err := filepath.Abs(command.Sequence)
		if err != nil {
			return fmt.Errorf("problem interpreting path %s\n%w", command.Sequence, err)
		}
		command.Sequence = absPath
	}
	return executeSequence(configPaths, valuesPaths, command.Sequence, command.Targets, command.Debug)
}

func executeSequence(configPaths []string, valuesPaths []string, sequencePath string, targets []string, debug bool) error {
	configObj, err := config.FromFilePaths(configPaths...)
	if err != nil {
		return err
	}

	configObj.Debug = command.Debug
	if configObj.Debug {
		log.SetLevel(log.DEBUG)
	} else {
		log.SetLevel(log.INFO)
	}

	if command.Json {
		log.SetLevel(log.SILENT)
		configObj.Json = true
	}

	valuesStore := kvstore.NewStore()
	for _, valuesPath := range valuesPaths {
		valuesBytes, err := os.ReadFile(valuesPath)
		if err != nil {
			return fmt.Errorf("unable to read values file at %s", valuesPath)
		}

		vTarget := map[string]any{}
		err = yaml.Unmarshal(valuesBytes, &vTarget)
		if err != nil {
			return fmt.Errorf("unable to parse yaml from values file at %s", valuesPath)
		}

		s, err := kvstore.FromMapping(vTarget)
		if err != nil {
			return fmt.Errorf("problem creating store from values file at %s\n%w", valuesPath, err)
		}

		valuesStore = valuesStore.Overlay(s)
	}

	// set the values storage object and carry it around here
	configObj.ValuesStore = valuesStore

	selectedHosts := map[string]struct{}{}
	if len(command.Targets) >= 1 && command.Targets[0] != "all" {
		for _, hostIdent := range command.Targets {
			selectedHosts[hostIdent] = struct{}{}
		}
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

	return executor.RunConcurrentExecutionGroup(command.Sequence, configObj, hostIdents)
}

func main() {
	for _, arg := range os.Args {
		if arg == "--version" {
			fmt.Printf("%s\n", Version)
			os.Exit(0)
		}
	}

	_ = kong.Parse(&command)
	err := run()
	if err != nil {
		log.Error(nil, "%s", err.Error())
		os.Exit(1)
	}
}
