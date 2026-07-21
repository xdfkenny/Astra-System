package prereq

import (
	"os/exec"
)

type DockerStatus int

const (
	DockerNotFound  DockerStatus = iota
	DockerNotRunning
	DockerOK
)

func CheckDocker() DockerStatus {
	if _, err := exec.LookPath("docker"); err != nil {
		return DockerNotFound
	}

	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	if err := cmd.Run(); err != nil {
		return DockerNotRunning
	}

	return DockerOK
}
