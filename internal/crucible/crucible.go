package crucible

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/executor"
	"github.com/frozengoats/crucible/internal/log"
	"github.com/frozengoats/kvstore"
	"github.com/goccy/go-yaml"
)

func ExecuteSequenceFromCwd(cwdPath string, extraConfigPaths []string, extraValuesPaths []string, sequencePath string, targets []string, debug bool, jsonOutput bool) error {
	mainConfigPath := filepath.Join(cwdPath, "config.yaml")
	_, err := os.Stat(mainConfigPath)
	if err != nil {
		return fmt.Errorf("no config.yaml could be located at %s, are you sure this is a crucible configuration?\n%w", mainConfigPath, err)
	}

	for i, configPath := range extraConfigPaths {
		if !filepath.IsAbs(configPath) {
			absPath, err := filepath.Abs(filepath.Join(cwdPath, configPath))
			if err != nil {
				return fmt.Errorf("problem interpreting path %s\n%w", configPath, err)
			}
			extraConfigPaths[i] = absPath
		}
	}
	configPaths := append([]string{mainConfigPath}, extraConfigPaths...)

	mainValuesPath := filepath.Join(cwdPath, "values.yaml")
	for i, valuesPath := range extraValuesPaths {
		if !filepath.IsAbs(valuesPath) {
			absPath, err := filepath.Abs(filepath.Join(cwdPath, valuesPath))
			if err != nil {
				return fmt.Errorf("problem interpreting path %s\n%w", valuesPath, err)
			}
			extraValuesPaths[i] = absPath
		}
	}

	var valuesPaths []string
	_, err = os.Stat(mainConfigPath)
	if err == nil {
		valuesPaths = append([]string{mainValuesPath}, extraValuesPaths...)
	} else {
		valuesPaths = extraValuesPaths
	}

	if !filepath.IsAbs(sequencePath) {
		absPath, err := filepath.Abs(sequencePath)
		if err != nil {
			return fmt.Errorf("problem interpreting path %s\n%w", sequencePath, err)
		}
		sequencePath = absPath
	}

	if len(targets) == 0 {
		return fmt.Errorf("must specify a deploy target, or `all` for all targets")
	}
	if len(targets) == 1 && targets[0] == "all" {
		targets = nil
	}
	return executeSequence(configPaths, valuesPaths, sequencePath, targets, debug, jsonOutput)
}

func executeSequence(configPaths []string, valuesPaths []string, sequencePath string, targets []string, debug bool, jsonOutput bool) error {
	configObj, err := config.FromFilePaths(configPaths...)
	if err != nil {
		return err
	}

	configObj.Debug = debug
	if configObj.Debug {
		log.SetLevel(log.DEBUG)
	} else {
		log.SetLevel(log.INFO)
	}

	if jsonOutput {
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
	if len(targets) == 0 {
		for _, hostIdent := range targets {
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

	return executor.RunConcurrentExecutionGroup(sequencePath, configObj, hostIdents)
}
