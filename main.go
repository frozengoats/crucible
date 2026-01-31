package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/frozengoats/crucible/internal/crucible"
	"github.com/frozengoats/crucible/internal/executor"
	"github.com/frozengoats/crucible/internal/log"
	"github.com/frozengoats/crucible/internal/oci"
	"golang.org/x/term"
)

var Version string = "dev"

type RemoveCmd struct {
	Url []string `arg:"" help:"url of recipe to remove from local cache, in the form of oci://<registry>/<repository>/<name>[:<tag>]"`
}

func (c *RemoveCmd) Run() error {
	return crucible.RemoveRecipes(c.Url...)
}

type ListCmd struct {
}

func (c *ListCmd) Run() error {
	return crucible.ListRecipes()
}

type DownloadCmd struct {
	Url   string `arg:"" help:"url of recipe to download, in the form of oci://<registry>/<repository>/<name>[:<tag>]"`
	Force bool   `short:"f" help:"force download, wiping any previous download first"`
}

func (c *DownloadCmd) Run() error {
	return crucible.DownloadRecipe(c.Url, c.Force)
}

type InitCmd struct {
	Name      string   `arg:"" help:"name of the recipe to initialize"`
	Sequences []string `short:"s" help:"list of explicit sequences to initialize"`
}

func (c *InitCmd) Run() error {
	return crucible.InitRecipe(c.Name, c.Sequences)
}

type LogoutCmd struct {
	Registry string `arg:"" help:"the registry domain"`
}

func (c *LogoutCmd) Run() error {
	credMap, err := oci.LoadCredentials()
	if err != nil {
		return err
	}

	delete(credMap, c.Registry)
	if err = oci.SaveCredentials(credMap); err != nil {
		return err
	}

	fmt.Printf("logged out of %s\n", c.Registry)
	return nil
}

type LoginCmd struct {
	Registry string `arg:"" help:"the registry domain"`
	Username string `arg:"" help:"the registry username"`
}

func (c *LoginCmd) Run() error {
	if !strings.HasPrefix(c.Registry, "http://") && !strings.HasPrefix(c.Registry, "oci://") {
		c.Registry = fmt.Sprintf("oci://%s", c.Registry)
	}
	u, err := url.Parse(c.Registry)
	if err != nil {
		return err
	}

	fmt.Printf("enter password/token for user %s: ", c.Username)
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	fmt.Printf("\n")

	credMap, err := oci.LoadCredentials()
	if err != nil {
		return err
	}

	hostname := strings.ToLower(u.Hostname())
	credMap[hostname] = &oci.Credentials{
		Username: c.Username,
		Password: pwBytes,
	}

	err = oci.SaveCredentials(credMap)
	if err != nil {
		return err
	}

	fmt.Printf("credential file updated for %s\n", hostname)

	return nil
}

type PublishCmd struct {
	RecipeDir string `short:"r" help:"change the recipe directory to this location (defaults to cwd)"`
	Registry  string `arg:"" help:"the full name of the OCI registry (excluding recipe name and version tag)"`
}

func (c *PublishCmd) Run() error {
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

	return crucible.PublishRecipe(cwd, c.Registry)
}

type RunCmd struct {
	RecipeDir string   `short:"r" help:"specify recipe directory or oci:// url (defaults to current directory)"`
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

	_, ok, err := crucible.LintRecipe(cwd)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("lint was unsuccessful")
	}

	return nil
}

var CLI struct {
	Init     InitCmd     `cmd:"" help:"initialize a new crucible recipe"`
	Run      RunCmd      `cmd:"" help:"run a crucible recipe"`
	Lint     LintCmd     `cmd:"" help:"lint a crucible recipe"`
	Info     InfoCmd     `cmd:"" help:"display recipe info"`
	Publish  PublishCmd  `cmd:"" help:"publish recipe to OCI registry"`
	Download DownloadCmd `cmd:"" help:"download recipe from OCI registry"`
	Login    LoginCmd    `cmd:"" help:"login to OCI registry"`
	List     ListCmd     `cmd:"" help:"list downloaded recipes"`
	Remove   RemoveCmd   `cmd:"" help:"remove recipe from local download cache"`
	Logout   LogoutCmd   `cmd:"" help:"logout of OCI registry"`
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
