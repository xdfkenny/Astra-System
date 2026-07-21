//go:build !windows

package prereq

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CheckDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("Docker not found on PATH")
	}
	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker daemon not running: %w", err)
	}
	return nil
}

func CheckWSL() error {
	return nil
}

func InstallWSL() error {
	return nil
}

func InstallDockerDesktop(silent bool) error {
	if !silent {
		fmt.Println("  → Downloading Docker...")
	}

	platform := "linux"
	arch := "amd64"
	if s, err := exec.Command("uname", "-s").Output(); err == nil {
		platform = strings.ToLower(strings.TrimSpace(string(s)))
	}
	if a, err := exec.Command("uname", "-m").Output(); err == nil {
		arch = strings.TrimSpace(string(a))
		if arch == "x86_64" {
			arch = "amd64"
		} else if arch == "aarch64" {
			arch = "arm64"
		}
	}

	if platform == "darwin" {
		url := fmt.Sprintf("https://desktop.docker.com/mac/main/%s/Docker.dmg", arch)
		tmpDir := os.TempDir()
		installerPath := filepath.Join(tmpDir, "Docker.dmg")
		if err := downloadFile(url, installerPath); err != nil {
			return fmt.Errorf("download Docker Desktop: %w", err)
		}
		fmt.Println("  → Docker.dmg downloaded to", installerPath)
		fmt.Println("  ! Open the .dmg and drag Docker to Applications.")
		fmt.Println("  ! Then start Docker and re-run this installer.")
		return nil
	}

	fmt.Println("  → On Linux, run: curl -fsSL https://get.docker.com | sudo sh")
	fmt.Println("  ! Then re-run this installer.")
	return fmt.Errorf("Docker not installed. See https://docs.docker.com/engine/install/")
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}
