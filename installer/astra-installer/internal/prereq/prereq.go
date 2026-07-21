package prereq

import (
	"os/exec"
	"time"
)

func DockerInstalled() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func WaitForDocker(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return nil
}
