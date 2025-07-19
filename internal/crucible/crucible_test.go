package crucible

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/frozengoats/crucible/internal/ssh"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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

var (
	CompletionFile string = filepath.Join(ContainerUnixSocketDir, "complete")
)

type CrucibleTestSuite struct {
	suite.Suite

	sshContainer      testcontainers.Container
	sshAgentContainer testcontainers.Container
	sshHost           string
	testDataDir       string
}

func (suite *CrucibleTestSuite) SetupTest() {
	testDataDir, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	suite.testDataDir = testDataDir
	suite.NoError(err)

	configScript := filepath.Join(testDataDir, "config.sh")
	pubKey := filepath.Join(testDataDir, "id_ed25519.pub")
	pubKeyPhrase := filepath.Join(testDataDir, "id_ed25519_passphrase.pub")
	suite.NoError(err)

	req := testcontainers.ContainerRequest{
		Image:           SshdImage,
		AlwaysPullImage: true,
		WaitingFor:      wait.ForFile("/home/test/done.file").WithStartupTimeout(10 * time.Second),
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      configScript,
				ContainerFilePath: "/etc/config/config.sh",
				FileMode:          0o555,
			},
			{
				HostFilePath:      pubKey,
				ContainerFilePath: "/tmp/id_ed25519.pub",
			},
			{
				HostFilePath:      pubKeyPhrase,
				ContainerFilePath: "/tmp/id_ed25519_passphrase.pub",
			},
		},
		ExposedPorts: []string{SshPort},
	}
	suite.sshContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.NoError(err)

	sshHost, err := suite.sshContainer.Host(context.Background())
	suite.NoError(err)

	sshPort, err := suite.sshContainer.MappedPort(context.Background(), nat.Port(SshPort))
	suite.NoError(err)

	suite.sshHost = fmt.Sprintf("%s:%s", sshHost, sshPort.Port())

	_ = os.RemoveAll(AgentUnixSocketDir)
	err = os.MkdirAll(AgentUnixSocketDir, 0777)
	suite.NoError(err)

	cUser, err := user.Current()
	suite.NoError(err)

	socketFile := filepath.Join(ContainerUnixSocketDir, "agent.sock")
	req = testcontainers.ContainerRequest{
		Image:           SshAgentImage,
		AlwaysPullImage: true,
		WaitingFor:      wait.ForFile(CompletionFile).WithStartupTimeout(10 * time.Second),
		Env: map[string]string{
			"COMPLETION_FILE": CompletionFile,
			"SSH_AUTH_SOCK":   socketFile,
		},
		Mounts: testcontainers.Mounts(testcontainers.BindMount(AgentUnixSocketDir, testcontainers.ContainerMountTarget(ContainerUnixSocketDir))),
		ConfigModifier: func(c *container.Config) {
			c.User = fmt.Sprintf("%s:%s", cUser.Uid, cUser.Gid)
		},
	}

	suite.sshAgentContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.NoError(err)

	err = ssh.InitAgentInstance(ssh.WithSshAuthSock(filepath.Join(AgentUnixSocketDir, "agent.sock")))
	suite.NoError(err)

	err = os.WriteFile(KnownHostsFile, []byte{}, 0600)
	suite.NoError(err)
	_, err = ssh.GetKnownHostsInstance(KnownHostsFile)
	suite.NoError(err)
}

func (suite *CrucibleTestSuite) TearDownTest() {
	if suite.sshContainer != nil {
		testcontainers.CleanupContainer(suite.T(), suite.sshContainer)
	}

	if suite.sshAgentContainer != nil {
		testcontainers.CleanupContainer(suite.T(), suite.sshAgentContainer)
	}

	_ = os.RemoveAll(AgentUnixSocketDir)
}

func (suite *CrucibleTestSuite) BasicTest() {

}

func TestCrucible(t *testing.T) {
	suite.Run(t, new(CrucibleTestSuite))
}
