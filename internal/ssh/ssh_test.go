package ssh

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

var dirName = "/tmp/rsync_test"

func touch(path string) error {
	return os.WriteFile(path, []byte{}, 0o777)
}

func prepTestEnvironment() error {
	_ = os.RemoveAll(dirName)
	err := os.MkdirAll(dirName, 0o777)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(dirName, "sub1_a", "sub2_a"), 0o777)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(dirName, "sub1_b", "sub2_a"), 0o777)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(dirName, "sub1_b", "sub2_b"), 0o777)
	if err != nil {
		return err
	}

	files := []string{
		filepath.Join(dirName, "file.txt"),
		filepath.Join(dirName, "sub1_a", "file.txt"),
		filepath.Join(dirName, "sub1_a", "file1.txt"),
		filepath.Join(dirName, "sub1_b", "sub2_b", "file.txt"),
	}
	for _, f := range files {
		err := touch(f)
		if err != nil {
			return err
		}
	}

	return nil
}

type SshTestSuite struct {
	suite.Suite

	sshContainer      testcontainers.Container
	sshAgentContainer testcontainers.Container
	sshHost           string
}

func (suite *SshTestSuite) SetupTest() {
	configScript, err := filepath.Abs(filepath.Join(".", "testdata", "config.sh"))
	suite.NoError(err)
	pubKey, err := filepath.Abs(filepath.Join(".", "testdata", "id_ed25519.pub"))
	suite.NoError(err)
	pubKeyPhrase, err := filepath.Abs(filepath.Join(".", "testdata", "id_ed25519_passphrase.pub"))
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

	err = InitAgentInstance(WithSshAuthSock(filepath.Join(AgentUnixSocketDir, "agent.sock")))
	suite.NoError(err)

	err = os.WriteFile(KnownHostsFile, []byte{}, 0600)
	suite.NoError(err)
	err = InitializeKnownHosts(WithKnownHostsFile(KnownHostsFile))
	suite.NoError(err)
}

func (suite *SshTestSuite) TearDownTest() {
	if suite.sshContainer != nil {
		testcontainers.CleanupContainer(suite.T(), suite.sshContainer)
	}

	if suite.sshAgentContainer != nil {
		testcontainers.CleanupContainer(suite.T(), suite.sshAgentContainer)
	}

	_ = os.RemoveAll(AgentUnixSocketDir)
}

func (suite *SshTestSuite) TestUnknownHostDontAllow() {
	privKey, err := filepath.Abs(filepath.Join(".", "testdata", "id_ed25519"))
	suite.NoError(err)

	sshSession := NewSsh(suite.sshHost, "test", privKey, WithPassphraseProviderOption(NewTypedPassphraseProvider()))
	defer sshSession.Close()

	// should fail b/c host is unknown and we don't allow for that
	err = sshSession.Connect()
	suite.Error(err)
}

func (suite *SshTestSuite) TestUnknownHostAllow() {
	privKey, err := filepath.Abs(filepath.Join(".", "testdata", "id_ed25519"))
	suite.NoError(err)

	// allow the unknown host to be connected to
	sshSession := NewSsh(suite.sshHost, "test", privKey, WithAllowUnknownHostsOption(true), WithPassphraseProviderOption(NewTypedPassphraseProvider()))
	defer sshSession.Close()

	err = sshSession.Connect()
	suite.NoError(err)
}

func (suite *SshTestSuite) TestKnownHostNoProblem() {
	privKey, err := filepath.Abs(filepath.Join(".", "testdata", "id_ed25519"))
	suite.NoError(err)

	// allow the unknown host to be connected to, so that we can cache it in our known_hosts
	sshSession := NewSsh(suite.sshHost, "test", privKey, WithAllowUnknownHostsOption(true), WithPassphraseProviderOption(NewTypedPassphraseProvider()))
	defer sshSession.Close()
	err = sshSession.Connect()
	suite.NoError(err)

	// this time, fail if a host is unknown.  it should already be part of our known_hosts though, so it should pass
	sshSession = NewSsh(suite.sshHost, "test", privKey, WithPassphraseProviderOption(NewTypedPassphraseProvider()))
	defer sshSession.Close()
	err = sshSession.Connect()
	suite.NoError(err)
}

func (suite *SshTestSuite) TestKeyWithPassphrase() {
	privKey, err := filepath.Abs(filepath.Join(".", "testdata", "id_ed25519_passphrase"))
	suite.NoError(err)

	// block entry with bad passphrase on locked private key
	sshSession := NewSsh(suite.sshHost, "test", privKey, WithAllowUnknownHostsOption(true), WithPassphraseProviderOption(NewDefaultPassphraseProvider("badphrase")))
	defer sshSession.Close()
	err = sshSession.Connect()
	suite.Error(err)

	// admit entry once by providing the passphrase when prompted with the correct passphrase
	sshSession = NewSsh(suite.sshHost, "test", privKey, WithAllowUnknownHostsOption(true), WithPassphraseProviderOption(NewDefaultPassphraseProvider(TestPassphrase)))
	defer sshSession.Close()
	err = sshSession.Connect()
	suite.NoError(err)

	// admit entry even with empty phrase, because agent should now hold the unlocked key
	sshSession = NewSsh(suite.sshHost, "test", privKey, WithAllowUnknownHostsOption(true), WithPassphraseProviderOption(NewDefaultPassphraseProvider("")))
	defer sshSession.Close()
	err = sshSession.Connect()
	suite.NoError(err)
}

func (suite *SshTestSuite) TestRsyncNoPassphrase() {
	// prepare ssh session
	privKey, err := filepath.Abs(filepath.Join(".", "testdata", "id_ed25519"))
	suite.NoError(err)

	// allow the unknown host to be connected to
	sshSession := NewSsh(suite.sshHost, "test", privKey, WithAllowUnknownHostsOption(true), WithPassphraseProviderOption(NewTypedPassphraseProvider()))
	defer sshSession.Close()

	err = sshSession.Connect()
	suite.NoError(err)
	err = sshSession.Close()
	suite.NoError(err)

	err = prepTestEnvironment()
	suite.NoError(err)

	err = Rsync("test", suite.sshHost, privKey, "/tmp/rsync_test", "/tmp/target", "-o", fmt.Sprintf("UserKnownHostsFile=%s", KnownHostsFile))
	suite.NoError(err)
}

func TestSshSuite(t *testing.T) {
	suite.Run(t, new(SshTestSuite))
}
