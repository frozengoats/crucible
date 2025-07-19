package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/frozengoats/crucible/internal/crucible"
	"github.com/frozengoats/crucible/internal/log"
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

	if len(command.Targets) == 0 {
		return fmt.Errorf("must specify a deploy target, or `all` for all targets")
	}
	if len(command.Targets) == 1 && command.Targets[0] == "all" {
		command.Targets = nil
	}
	return crucible.ExecuteSequence(configPaths, valuesPaths, command.Sequence, command.Targets, command.Debug, command.Json)
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
