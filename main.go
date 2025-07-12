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

var command struct {
	Cwd      string   `short:"c" help:"change the current working directory to this location"`
	Configs  []string `short:"s" help:"list of paths to any config yaml overrides, stackable in order of occurrence (excluding config.yaml)"`
	Values   []string `short:"v" help:"list of paths to values files, stackable in order of occurrence (excluding values.yaml)"`
	Sequence string   `arg:"" help:"the full or relative path to the sequence to execute"`
	Targets  []string `arg:"" help:"named machine targets and/or groups against which to execute the sequence"`
	Debug    bool     `short:"d" help:"enable debug mode"`
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

	configPaths := append([]string{configPath}, command.Configs...)
	configObj, err := config.FromFilePaths(cwd, configPaths...)
	if err != nil {
		return err
	}

	configObj.Debug = command.Debug

	var valuesStore *kvstore.Store
	valuesStack := []string{}
	valuesPath := filepath.Join(cwd, "values.yaml")
	_, err = os.Stat(valuesPath)
	if err != nil {
		valuesStore = kvstore.NewStore()
		valuesStack = command.Values
	} else {
		valuesStack = append(valuesStack, valuesPath)
		valuesStack = append(valuesStack, command.Values...)
	}

	for _, vp := range valuesStack {
		if !filepath.IsAbs(vp) {
			nvp, err := filepath.Abs(filepath.Join(cwd, vp))
			if err != nil {
				return fmt.Errorf("unable to reconcile values file %s", vp)
			}
			vp = nvp
		}

		valuesBytes, err := os.ReadFile(vp)
		if err != nil {
			return fmt.Errorf("unable to read values file at %s", vp)
		}

		vTarget := map[string]any{}
		err = yaml.Unmarshal(valuesBytes, &vTarget)
		if err != nil {
			return fmt.Errorf("unable to parse yaml from values file at %s", vp)
		}

		s, err := kvstore.FromMapping(vTarget)
		if err != nil {
			return fmt.Errorf("problem creating store from values file at %s\n%w", vp, err)
		}

		valuesStore = valuesStore.Overlay(s)
	}

	// set the values storage object and carry it around here
	configObj.ValuesStore = valuesStore

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

	return executor.RunConcurrentExecutionGroup(command.Sequence, configObj, hostIdents)
}

func main() {
	_ = kong.Parse(&command)
	err := run()
	if err != nil {
		log.Error(nil, "%s", err.Error())
		os.Exit(1)
	}
}
