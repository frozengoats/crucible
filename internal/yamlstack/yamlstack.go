package yamlstack

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

// stackMap stacks top on top of base, modifying base in the process
// elements from top are always shallow copied onto base, thus they must not be mutated later
func stackMap(base map[string]any, top map[string]any, fullKeyPath []string) error {
	for key, value := range top {
		target, ok := base[key]
		if !ok {
			base[key] = value
			continue
		}

		switch baseTarget := target.(type) {
		case map[string]any:
			switch topTarget := value.(type) {
			case map[string]any:
				// both base and top are maps, continue traversal
				err := stackMap(baseTarget, topTarget, append(fullKeyPath, key))
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("top layer changes data type at key %s", strings.Join(append(fullKeyPath, key), "."))
			}
		default:
			base[key] = value
		}
	}

	return nil
}

func StackYaml(stackPaths ...string) ([]byte, error) {
	base := map[string]any{}

	for _, stackPath := range stackPaths {
		stackBytes, err := os.ReadFile(stackPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read file %s\n%w", stackPath, err)
		}

		current := map[string]any{}
		err = yaml.Unmarshal(stackBytes, &current)
		if err != nil {
			return nil, fmt.Errorf("problem parsing yaml in %s\n%w", stackPath, err)
		}

		err = stackMap(base, current, nil)
		if err != nil {
			return nil, fmt.Errorf("problem stacking config values: %w", err)
		}
	}

	return yaml.Marshal(&base)
}
