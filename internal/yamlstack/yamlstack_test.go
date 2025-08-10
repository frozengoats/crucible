package yamlstack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapStackerTypeChange(t *testing.T) {
	base := map[string]any{
		"hello": []string{"abc", "def"},
	}

	top := map[string]any{
		"hello": []int{1, 2, 3},
	}

	err := stackMap(base, top, nil)
	assert.NoError(t, err)

	switch base["hello"].(type) {
	case []int:
	default:
		assert.Fail(t, "value should be overwritten with an []int")
	}
}

func TestMapStackMapChangeType(t *testing.T) {
	base := map[string]any{
		"hello": map[string]any{
			"duck":  1,
			"goose": 2,
		},
	}

	top := map[string]any{
		"hello": []int{1, 2, 3},
	}

	err := stackMap(base, top, nil)
	// maps should never just change type, this forbids merging
	assert.Error(t, err)
}

func TestMapStackMapMerge(t *testing.T) {
	base := map[string]any{
		"hello": map[string]any{
			"duck":  1,
			"goose": 2,
		},
	}

	top := map[string]any{
		"hello": map[string]any{
			"dog":   4,
			"goose": 24,
		},
	}

	err := stackMap(base, top, nil)
	// maps should never just change type, this forbids merging
	assert.NoError(t, err)
	assert.Len(t, base["hello"], 3)
	assert.Equal(t, base["hello"].(map[string]any)["goose"], 24)
}
