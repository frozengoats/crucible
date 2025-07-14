package ssh

import (
	"fmt"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

type SshInfo struct {
	User           string
	KeyPath        string
	KnownHostsPath string
	Hostname       string
	Port           int
}

func GetSshInfo(hostAlias string) (*SshInfo, error) {
	sshInfo := &SshInfo{}

	cmd := exec.Command("ssh", "-G", hostAlias)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("unable to get ssh info for %s: %w", hostAlias, err)
	}

	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "user":
			sshInfo.User = value
		case "identityfile":
			sshInfo.KeyPath = value
		case "userknownhostsfile":
			sshInfo.KnownHostsPath = value
		case "hostname":
			sshInfo.Hostname = value
		case "port":
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("unable to parse port from ssh configuration for %s: %w", hostAlias, err)
			}
			sshInfo.Port = int(i)
		}
	}

	if strings.Contains(sshInfo.KeyPath, "~") || strings.Contains(sshInfo.KnownHostsPath, "~") {
		u, err := user.Lookup(sshInfo.User)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve user %s when getting ssh info for %s: %w", sshInfo.User, hostAlias, err)
		}

		sshInfo.KeyPath = strings.ReplaceAll(sshInfo.KeyPath, "~", u.HomeDir)
		sshInfo.KnownHostsPath = strings.ReplaceAll(sshInfo.KnownHostsPath, "~", u.HomeDir)
	}

	// when a host provided by the user contains a port number, allow this port number to supercede the one
	// provided by the ssh config
	if strings.Contains(sshInfo.Hostname, ":") {
		parts := strings.SplitN(sshInfo.Hostname, ":", 2)
		sshInfo.Hostname = parts[0]
		i, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse port from ssh configuration for %s: %w", hostAlias, err)
		}
		sshInfo.Port = int(i)
	}

	return sshInfo, nil
}
