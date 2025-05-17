package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/frozengoats/crucible/internal/config"
)

var command struct {
	Cwd         string   `short:"c" help:"change the current working directory to this location"`
	ConfigStack []string `short:"s" help:"list of paths to any config yaml overrides, stackable in order of occurrence"`
	Sequence    string   `arg:"" help:"the full or relative path to the sequence to execute"`
	Targets     []string `arg:"" help:"named machine targets and/or groups against which to execute the sequence"`
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

	fmt.Println(configObj)

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
