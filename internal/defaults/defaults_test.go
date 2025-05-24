package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type HostExample struct {
	Host     string
	NumConns int `default:"4"`
}

type NestedStruct struct {
	Name  string `default:"what"`
	Value string
}

type AnotherStruct struct {
	FullName string `default:"what"`
	Value    string
	Hosts    map[string]*HostExample
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

	x.NestedOther.Hosts = map[string]*HostExample{}
	x.NestedOther.Hosts["x"] = &HostExample{
		Host: "bob",
	}

	err := ApplyDefaults(x)
	assert.NoError(t, err)
	assert.Equal(t, "hello", x.Name)
	assert.Nil(t, x.NestedNil)
	assert.Equal(t, "what", x.NestedOther.FullName)
	assert.Equal(t, 4, x.NestedOther.Hosts["x"].NumConns)
}
