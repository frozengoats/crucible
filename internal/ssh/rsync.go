package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Rsync(username string, host string, port int, keyFile string, source string, dest string, sshParams ...string) error {
	if port != 22 {
		sshParams = append(sshParams, "-p", fmt.Sprintf("%d", port))
	}

	sshCommand := []string{
		"ssh",
		"-i",
		keyFile,
	}
	sshCommand = append(sshCommand, sshParams...)

	sshString := strings.Join(sshCommand, " ")
	command := []string{
		"rsync", "-av", "-e", sshString, source, fmt.Sprintf("%s@%s:%s", username, host, dest),
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		quoted := []string{}
		for _, c := range command {
			if strings.Contains(c, " ") {
				quoted = append(quoted, fmt.Sprintf("\"%s\"", c))
			} else {
				quoted = append(quoted, c)
			}
		}
		return fmt.Errorf("%s\n%w", strings.Join(quoted, " "), err)
	}

	return nil
}
