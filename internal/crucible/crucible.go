package crucible

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/executor"
	"github.com/frozengoats/crucible/internal/log"
	"github.com/frozengoats/kvstore"
	"github.com/goccy/go-yaml"
)

func ExecuteSequenceFromCwd(cwdPath string, extraConfigPaths []string, extraValuesPaths []string, sequencePath string, targets []string, debug bool, jsonOutput bool) ([]byte, error) {
	mainConfigPath := filepath.Join(cwdPath, "config.yaml")
	_, err := os.Stat(mainConfigPath)
	if err != nil {
		return nil, fmt.Errorf("no config.yaml could be located at %s, are you sure this is a crucible configuration?\n%w", mainConfigPath, err)
	}

	for i, configPath := range extraConfigPaths {
		if !filepath.IsAbs(configPath) {
			absPath, err := filepath.Abs(filepath.Join(cwdPath, configPath))
			if err != nil {
				return nil, fmt.Errorf("problem interpreting path %s\n%w", configPath, err)
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
				return nil, fmt.Errorf("problem interpreting path %s\n%w", valuesPath, err)
			}
			extraValuesPaths[i] = absPath
		}
	}

	var valuesPaths []string
	_, err = os.Stat(mainValuesPath)
	if err == nil {
		valuesPaths = append([]string{mainValuesPath}, extraValuesPaths...)
	} else {
		valuesPaths = extraValuesPaths
	}

	if !filepath.IsAbs(sequencePath) {
		absPath, err := filepath.Abs(filepath.Join(cwdPath, sequencePath))
		if err != nil {
			return nil, fmt.Errorf("problem interpreting path %s\n%w", sequencePath, err)
		}
		sequencePath = absPath
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("must specify a deploy target, or `all` for all targets")
	}
	if len(targets) == 1 && targets[0] == "all" {
		targets = nil
	}
	return executeSequence(cwdPath, configPaths, valuesPaths, sequencePath, targets, debug, jsonOutput)
}

func executeSequence(cwdPath string, configPaths []string, valuesPaths []string, sequencePath string, targets []string, debug bool, jsonOutput bool) ([]byte, error) {
	configObj, err := config.FromFilePaths(configPaths...)
	if err != nil {
		return nil, err
	}

	configObj.CwdPath = cwdPath
	configObj.Debug = debug
	if configObj.Debug {
		log.SetLevel(log.DEBUG)
	} else {
		log.SetLevel(log.INFO)
	}

	// fix paths containing the home directory
	if strings.Contains(configObj.Executor.Ssh.KeyPath, "~") {
		configObj.Executor.Ssh.KeyPath = strings.ReplaceAll(configObj.Executor.Ssh.KeyPath, "~", configObj.User.HomeDir)
	}
	if strings.Contains(configObj.Executor.Ssh.KnownHostsPath, "~") {
		configObj.Executor.Ssh.KnownHostsPath = strings.ReplaceAll(configObj.Executor.Ssh.KnownHostsPath, "~", configObj.User.HomeDir)
	}
	for _, host := range configObj.Hosts {
		if strings.Contains(host.Ssh.KeyPath, "~") {
			host.Ssh.KeyPath = strings.ReplaceAll(host.Ssh.KeyPath, "~", configObj.User.HomeDir)
		}
	}

	if jsonOutput {
		log.SetLevel(log.SILENT)
		configObj.Json = true
	}

	valuesStore := kvstore.NewStore()
	for _, valuesPath := range valuesPaths {
		valuesBytes, err := os.ReadFile(valuesPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read values file at %s", valuesPath)
		}

		vTarget := map[string]any{}
		err = yaml.Unmarshal(valuesBytes, &vTarget)
		if err != nil {
			return nil, fmt.Errorf("unable to parse yaml from values file at %s", valuesPath)
		}

		s, err := kvstore.FromMapping(vTarget)
		if err != nil {
			return nil, fmt.Errorf("problem creating store from values file at %s\n%w", valuesPath, err)
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
		return nil, fmt.Errorf("no hosts specified")
	}

	config.ConfigInst = configObj

	return executor.RunConcurrentExecutionGroup(sequencePath, configObj, hostIdents)
}
