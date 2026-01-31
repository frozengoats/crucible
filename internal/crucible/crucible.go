package crucible

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/frozengoats/crucible/internal/config"
	"github.com/frozengoats/crucible/internal/executor"
	"github.com/frozengoats/crucible/internal/log"
	"github.com/frozengoats/crucible/internal/oci"
	"github.com/frozengoats/crucible/internal/sequence"
	"github.com/frozengoats/kvstore"
	"github.com/goccy/go-yaml"
)

type Recipe struct {
	Version     string            `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Sequences   map[string]string `yaml:"sequences"`
}

var (
	seqNameValidator    = regexp.MustCompile(`^[a-z]+$`)
	recipeNameValidator = regexp.MustCompile(`^[a-z][a-z_0-9]*$`)
	versionValidator    = regexp.MustCompile(`^\d\.\d\.\d(\.[0-9a-z]+)?$`)
)

func validateVersion(versionString string) bool {
	return versionValidator.MatchString(versionString)
}

func (r *Recipe) Lint(recipePath string) (bool, error) {
	lintOk := true

	if r.Name == "" {
		lintOk = false
		log.Info(nil, "recipe name is not specified")
	} else {
		ok := recipeNameValidator.Match([]byte(r.Name))
		if !ok {
			lintOk = false
			log.Info(nil, "recipes name can only contain lowercase alphanumeric characters or underscores and must begin with an alpha character")
		}
	}

	if r.Description == "" {
		lintOk = false
		log.Info(nil, "recipe description is not specified")
	}

	if r.Version == "" {
		lintOk = false
		log.Info(nil, "recipe version is not specified")
	} else {
		isOk := validateVersion(r.Version)
		if !isOk {
			log.Info(nil, "recipe version must follow semantic versioning style (eg. <maj>.<min>.<patch>[.<extra>])")
		}
	}

	if len(r.Sequences) == 0 {
		lintOk = false
		log.Info(nil, "recipe has no public sequences defined")
	} else {
		for name, seqPath := range r.Sequences {
			if !seqNameValidator.MatchString(name) {
				lintOk = false
				log.Info(nil, "sequence name %s contains characters beyond lower-cased letters", name)
			}

			fullSeqPath := filepath.Join(recipePath, seqPath)
			_, err := os.Stat(fullSeqPath)
			if err != nil {
				return false, fmt.Errorf("sequence %s pointed to bad path %s: %w", name, fullSeqPath, err)
			}

			s, err := sequence.LoadSequence(recipePath, seqPath)
			if err != nil {
				return false, err
			}

			ok, err := s.Lint(recipePath)
			if err != nil {
				return false, fmt.Errorf("sequence at %s contained an error", seqPath)
			}
			if !ok {
				lintOk = false
			}
		}
	}

	if lintOk {
		log.Info(nil, "lint of \"%s\" was successful", recipePath)
	}

	return lintOk, nil
}

func PublishRecipe(cwdPath string, registryPrefix string) error {
	recipe, isLinted, err := LintRecipe(cwdPath)
	if err != nil {
		return err
	}
	if !isLinted {
		return fmt.Errorf("recipe does not pass linter, please run `crucible lint` and apply the corrective suggestions prior to publishing")
	}

	desc, err := oci.Publish(cwdPath, registryPrefix, recipe.Name, recipe.Version)
	if err != nil {
		return err
	}

	log.Info(nil, "recipe published and available at %s", desc.Url())
	return nil
}

func InitRecipe(name string, sequenceNames []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	name = strings.ToLower(name)
	recipeDir := filepath.Join(cwd, name)
	_, err = os.Stat(recipeDir)
	if err == nil {
		return fmt.Errorf("something already exists with the name: %s", name)
	}

	err = os.Mkdir(recipeDir, 0770)
	if err != nil {
		return err
	}

	if len(sequenceNames) == 0 {
		sequenceNames = append(sequenceNames, "myseq")
	}

	recipe := &Recipe{
		Version:     "0.0.1",
		Name:        name,
		Description: "my new recipe",
		Sequences:   map[string]string{},
	}

	seqsDir := filepath.Join(recipeDir, "sequences")
	err = os.Mkdir(seqsDir, 0770)
	if err != nil {
		return err
	}

	for _, seqName := range sequenceNames {
		seqName = strings.ToLower(seqName)
		seqPath := filepath.Join(seqsDir, seqName)

		recipe.Sequences[seqName] = filepath.Join("sequences", seqName)

		seq := &sequence.Sequence{
			Name:        seqName,
			Description: "it does this",
			Sequence:    []*sequence.Action{},
		}
		seqBytes, err := yaml.Marshal(seq)
		if err != nil {
			return err
		}

		err = os.WriteFile(seqPath, seqBytes, 0660)
		if err != nil {
			return err
		}
	}

	recipeBytes, err := yaml.Marshal(recipe)
	if err != nil {
		return err
	}
	recipePath := filepath.Join(recipeDir, "recipe.yaml")
	err = os.WriteFile(recipePath, recipeBytes, 0660)
	if err != nil {
		return err
	}

	valuesPath := filepath.Join(recipeDir, "values.yaml")
	err = os.WriteFile(valuesPath, []byte{}, 0660)
	if err != nil {
		return err
	}

	fmt.Printf("initialized recipe \"%s\"\n", name)

	return nil
}

func RemoveRecipe(url string) error {
	imageDescriptor, err := oci.NewImageDescriptor(url)
	if err != nil {
		return err
	}

	storagePath, err := imageDescriptor.StoragePath()
	if err != nil {
		return err
	}

	if err = os.RemoveAll(storagePath); err != nil {
		return err
	}

	fmt.Printf("%s removed from local download cache\n", url)
	return nil
}

func ListRecipes() error {
	baseDir, err := oci.GetOciStoragePath()
	if err != nil {
		return err
	}
	return filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Name() != "digest.sha" {
			return nil
		}

		dir, _ := filepath.Split(path)
		relPath, err := filepath.Rel(baseDir, dir)
		if err != nil {
			return err
		}
		repo, tag := filepath.Split(relPath)
		repo = strings.TrimSuffix(repo, "/")

		fmt.Printf("oci://%s:%s\n", repo, tag)

		return nil
	})
}

func LintRecipe(cwdPath string) (*Recipe, bool, error) {
	recipePath := filepath.Join(cwdPath, "recipe.yaml")
	_, err := os.Stat(recipePath)
	if err != nil {
		return nil, false, fmt.Errorf("unable to locate recipe.yaml, are you sure this is a crucible recipe?")
	}

	rBytes, err := os.ReadFile(recipePath)
	if err != nil {
		return nil, false, err
	}

	recipe := &Recipe{}
	err = yaml.UnmarshalWithOptions(rBytes, recipe, yaml.DisallowUnknownField())
	if err != nil {
		return nil, false, err
	}

	isLinted, err := recipe.Lint(cwdPath)
	return recipe, isLinted, err
}

func RecipeInfo(cwdPath string) error {
	recipePath := filepath.Join(cwdPath, "recipe.yaml")
	_, err := os.Stat(recipePath)
	if err != nil {
		return fmt.Errorf("unable to locate recipe.yaml, are you sure this is a crucible recipe?")
	}

	rBytes, err := os.ReadFile(recipePath)
	if err != nil {
		return err
	}

	recipe := &Recipe{}
	err = yaml.Unmarshal(rBytes, recipe)
	if err != nil {
		return err
	}

	fmt.Printf("Recipe: %s\n", recipe.Name)
	fmt.Printf("%s\n\n", recipe.Description)
	for s, sPath := range recipe.Sequences {
		fmt.Printf("Sequence: %s\n", s)
		fData, err := os.ReadFile(filepath.Join(cwdPath, sPath))
		if err != nil {
			fmt.Printf("error processing sequence: %s\n\n", err.Error())
			continue
		}
		seq := &sequence.Sequence{}
		err = yaml.Unmarshal(fData, seq)
		if err != nil {
			fmt.Printf("error processing sequence: %s\n\n", err.Error())
			continue
		}

		fmt.Printf("%s\n\n", seq.Description)
	}

	return nil
}

func DownloadRecipe(url string, force bool) error {
	imageDescriptor, err := oci.NewImageDescriptor(url)
	if err != nil {
		return err
	}

	return oci.Download(imageDescriptor, force)
}

func ExecuteSequenceFromCwd(cwdPath string, extraConfigPaths []string, extraValuesPaths []string, sequence string, targets []string, debug bool, jsonOutput bool) ([]byte, error) {
	if oci.IsOciUrl(cwdPath) {
		imageDescriptor, err := oci.NewImageDescriptor(cwdPath)
		if err != nil {
			return nil, err
		}

		digest, err := imageDescriptor.GetDigest()
		if err != nil {
			return nil, err
		}

		if digest == "" {
			if err = oci.Download(imageDescriptor, false); err != nil {
				return nil, err
			}
		}

		// rewrite the cwd to the downloaded storage path
		cwdPath, err = imageDescriptor.StoragePath()
		if err != nil {
			return nil, err
		}
	}

	recipePath := filepath.Join(cwdPath, "recipe.yaml")
	_, err := os.Stat(recipePath)
	if err != nil {
		return nil, fmt.Errorf("unable to locate recipe.yaml, are you sure this is a crucible recipe?")
	}

	recBytes, err := os.ReadFile(recipePath)
	if err != nil {
		return nil, fmt.Errorf("unable to load recipe file: %w", err)
	}
	recipe := &Recipe{}
	err = yaml.Unmarshal(recBytes, recipe)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal recipe data: %w", err)
	}

	if len(recipe.Sequences) == 0 {
		return nil, fmt.Errorf("no public sequences exist for this recipe")
	}

	seqPathTail, ok := recipe.Sequences[sequence]
	if !ok {
		return nil, fmt.Errorf("sequence \"%s\" does not exist", sequence)
	}
	sequencePath := filepath.Join(cwdPath, seqPathTail)

	if len(extraConfigPaths) == 0 {
		configYamlPath := filepath.Join(cwdPath, "config.yaml")
		_, err := os.Stat(configYamlPath)
		if err != nil {
			return nil, fmt.Errorf("you must provide a config.yaml, either in the root of your crucible recipe, or by supplying its location via flag")
		}
		extraConfigPaths = append(extraConfigPaths, configYamlPath)
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
	return executeSequence(cwdPath, extraConfigPaths, valuesPaths, sequencePath, targets, debug, jsonOutput)
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

	return executor.RunConcurrentExecutionGroup(sequencePath, configObj, hostIdents)
}
