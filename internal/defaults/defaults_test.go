package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type NestedStruct struct {
	Name  string `default:"what"`
	Value string
}

type AnotherStruct struct {
	FullName string `default:"what"`
	Value    string
}

type MainStruct struct {
	Name         string `default:"hello"`
	Value        string
	NestedNil    *NestedStruct
	NestedNotNil *NestedStruct
	NestedOther  AnotherStruct
}

func TestDefaults(t *testing.T) {
	x := &MainStruct{
		NestedNotNil: &NestedStruct{},
	}
	err := ApplyDefaults(x)
	assert.NoError(t, err)
	assert.Equal(t, "hello", x.Name)
	assert.Nil(t, x.NestedNil)
	assert.Equal(t, "what", x.NestedOther.FullName)
}
