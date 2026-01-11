package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/frozengoats/crucible/internal/crucible"
	"github.com/frozengoats/crucible/internal/executor"
	"github.com/frozengoats/crucible/internal/log"
)

var Version string = "dev"

type InitCmd struct {
	Name      string   `arg:"" help:"name of the recipe to initialize"`
	Sequences []string `short:"s" help:"list of explicit sequences to initialize"`
}

func (c *InitCmd) Run() error {
	return crucible.InitRecipe(c.Name, c.Sequences)
}

type RunCmd struct {
	RecipeDir string   `short:"r" help:"change the recipe directory to this location (defaults to cwd)"`
	Configs   []string `short:"c" help:"list of paths to any config yaml overrides, stackable in order of occurrence"`
	Values    []string `short:"v" help:"list of paths to values files, stackable in order of occurrence (excluding values.yaml)"`
	Sequence  string   `arg:"" help:"the name of the sequence to execute"`
	Targets   []string `arg:"" help:"named machine targets and/or groups against which to execute the sequence (\"all\" for all targets)"`
	Debug     bool     `short:"d" help:"enable debug mode"`
	Version   bool     `help:"display the current version"`
	Json      bool     `short:"j" help:"output results in json format, suppress normal logging"`
}

type InfoCmd struct {
	RecipeDir string
}

var LogErrors bool = true

func (c *InfoCmd) Run() error {
	var (
		cwd string
		err error
	)

	if c.RecipeDir != "" {
		cwd = c.RecipeDir
	} else {
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	return crucible.RecipeInfo(cwd)
}

func (c *RunCmd) Run() error {
	var (
		cwd string
		err error
	)

	if c.Json {
		// disable error logging to the stdout in this particular instance
		LogErrors = false
	}

	if c.RecipeDir != "" {
		cwd = c.RecipeDir
	} else {
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	jsonResult, err := crucible.ExecuteSequenceFromCwd(cwd, c.Configs, c.Values, c.Sequence, c.Targets, c.Debug, c.Json)
	if c.Json {
		if jsonResult == nil {
			r := executor.ResultObj{
				Error:        err.Error(),
				SuccessHosts: []string{},
				FailHosts:    []*executor.FailedHost{},
			}
			rBytes, err := json.Marshal(r)
			if err != nil {
				return err
			}
			fmt.Println(string(rBytes))
		} else {
			fmt.Println(string(jsonResult))
		}
	}
	return err
}

type LintCmd struct {
	RecipeDir string `short:"r" help:"change the recipe directory to this location (defaults to cwd)"`
}

func (c *LintCmd) Run() error {
	var (
		cwd string
		err error
	)

	if c.RecipeDir != "" {
		cwd = c.RecipeDir
	} else {
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	ok, err := crucible.LintRecipe(cwd)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("lint was unsuccessful")
	}

	return nil
}

var CLI struct {
	Init InitCmd `cmd:"" help:"initialize a new crucible recipe"`
	Run  RunCmd  `cmd:"" help:"run a crucible recipe"`
	Lint LintCmd `cmd:"" help:"lint a crucible recipe"`
	Info InfoCmd `cmd:"" help:"display recipe info"`
}

func main() {
	for _, arg := range os.Args {
		if arg == "--version" {
			fmt.Printf("%s\n", Version)
			os.Exit(0)
		}
	}

	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	if err != nil {
		if LogErrors {
			log.Error(nil, "%s", err.Error())
		}
		os.Exit(1)
	}
}
