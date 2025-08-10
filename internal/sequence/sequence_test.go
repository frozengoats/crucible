package sequence

import (
	"testing"

	"github.com/frozengoats/crucible/internal/cmdsession"
	"github.com/frozengoats/crucible/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestSequenceIteration(t *testing.T) {
	seq := &Sequence{
		Description: "top level",
		Sequence: []*Action{
			{
				Name:        "first",
				Description: "random task",
			},
			{
				Name:        "second",
				Description: "random task",
			},
			{
				Name:        "third as sub",
				Description: "subsequence",
				SubSequence: &Sequence{
					Description: "sub sequence",
					Sequence: []*Action{
						{
							Name:        "sub-first",
							Description: "random task",
						},
						{
							Name:        "sub-second",
							Description: "sub second",
						},
						{
							Name:        "sub third",
							Description: "another sub sequence",
							SubSequence: &Sequence{
								Description: "inner last",
								Sequence: []*Action{
									{
										Name:        "inner-sub-first",
										Description: "random task",
									},
								},
							},
						},
					},
				},
			},
			{
				Name:        "fourth",
				Description: "random task",
			},
		},
	}

	exInst := seq.NewExecutionInstance(
		cmdsession.NewDummyExecutionClient(),
		&config.Config{
			Hosts: map[string]*config.HostConfig{
				"testhost": {},
			},
		},
		"testhost",
	)

	totalActions := 0
	for {
		hasMore := exInst.HasMore()
		action, err := exInst.Next()
		assert.NoError(t, err)

		if hasMore {
			assert.NotNil(t, action)
		}
		if action == nil {
			break
		}
		totalActions++
	}

	assert.Equal(t, 6, totalActions)
	assert.False(t, exInst.HasMore())
}
