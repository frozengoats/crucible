package main

import (
	"fmt"
	"os"

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

	jsonResult, err := crucible.ExecuteSequenceFromCwd(cwd, command.Configs, command.Values, command.Sequence, command.Targets, command.Debug, command.Json)
	if command.Json {
		fmt.Println(string(jsonResult))
	}
	return err
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
