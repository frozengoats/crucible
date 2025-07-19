package crucible

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

const (
	SshdImage              string = "frozengoats/sshd:0"
	SshAgentImage          string = "frozengoats/ssh-agent:0"
	AgentUnixSocketDir     string = "/tmp/sshtest"
	ContainerUnixSocketDir string = "/etc/sshtest"
	SshPort                string = "22"
	KnownHostsFile         string = "/tmp/sshtest/known_hosts"
	TestPassphrase         string = "testphrase"
)

type CrucibleTestSuite struct {
	suite.Suite

	sshContainer      testcontainers.Container
	sshAgentContainer testcontainers.Container
	sshHost           string
}

func (suite *CrucibleTestSuite) SetupTest() {
}

func (suite *CrucibleTestSuite) TearDownTest() {
}

func (suite *CrucibleTestSuite) BasicTest() {
}

func TestCrucible(t *testing.T) {
	suite.Run(t, new(CrucibleTestSuite))
}
