package prereq

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func CheckDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("Docker not found on PATH")
	}

	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Docker daemon not running: %w", err)
	}
	fmt.Printf("  ✓ Docker Engine %s", strings.TrimSpace(string(out)))
	return nil
}

func InstallDockerDesktop(silent bool) error {
	if !silent {
		fmt.Println("  → Downloading Docker Desktop for Windows...")
	}

	url := "https://desktop.docker.com/win/stable/Docker%20Desktop%20Installer.exe"
	tmpDir := os.TempDir()
	installerPath := filepath.Join(tmpDir, "DockerDesktopInstaller.exe")

	if err := downloadFile(url, installerPath); err != nil {
		return fmt.Errorf("download Docker Desktop: %w", err)
	}

	if !silent {
		fmt.Println("  → Installing Docker Desktop (this may take several minutes)...")
		fmt.Println("  ! A system restart may be required after installation")
	}

	cmd := exec.Command(installerPath, "install", "--quiet", "--accept-license")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install Docker Desktop: %w", err)
	}

	return nil
}

func WaitForDocker(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("Docker did not start within %v", timeout)
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
