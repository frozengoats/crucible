package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSemverEqual1(t *testing.T) {
	v1, err := semver("v5.0.2")
	assert.NoError(t, err)
	v2, err := semver("5.0.2")
	assert.NoError(t, err)
	assert.Equal(t, v1, v2)
}

func TestSemverEqual2(t *testing.T) {
	v1, err := semver("v5")
	assert.NoError(t, err)
	v2, err := semver("5.0.0")
	assert.NoError(t, err)
	assert.Equal(t, v1, v2)
}

func TestSemverEqual3(t *testing.T) {
	v1, err := semver("v5.0")
	assert.NoError(t, err)
	v2, err := semver("5.0.0")
	assert.NoError(t, err)
	assert.Equal(t, v1, v2)
}

func TestSemverGreater1(t *testing.T) {
	v1, err := semver("5.0.1")
	assert.NoError(t, err)
	v2, err := semver("5.0.0")
	assert.NoError(t, err)
	assert.Greater(t, v1, v2)
}

func TestSemverGreater2(t *testing.T) {
	v1, err := semver("5.0.1")
	assert.NoError(t, err)
	v2, err := semver("5.0.0.alpha")
	assert.NoError(t, err)
	assert.Greater(t, v1, v2)
}

func TestSemverGreater3(t *testing.T) {
	v1, err := semver("5.10")
	assert.NoError(t, err)
	v2, err := semver("5.9.0")
	assert.NoError(t, err)
	assert.Greater(t, v1, v2)
}

func TestSemverGreater4(t *testing.T) {
	v1, err := semver("5.0.0")
	assert.NoError(t, err)
	v2, err := semver("04.10.10.10")
	assert.NoError(t, err)
	assert.Greater(t, v1, v2)
}
