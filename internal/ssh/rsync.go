package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Rsync(username string, host string, keyFile string, source string, dest string, params ...string) error {
	parts := strings.Split(host, ":")
	hostname := parts[0]
	sshCommand := []string{
		"ssh",
		"-i",
		keyFile,
	}

	if len(parts) > 1 {
		sshCommand = append(sshCommand, "-p", parts[1])
	}

	sshCommand = append(sshCommand, params...)

	sshString := strings.Join(sshCommand, " ")
	cmd := exec.Command("rsync", "-e", sshString, source, fmt.Sprintf("%s@%s:%s", username, hostname, dest))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
