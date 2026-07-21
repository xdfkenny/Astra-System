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
	fmt.Printf("  ✓ Docker Engine %s\n", strings.TrimSpace(string(out)))
	return nil
}

func CheckWSL() error {
	cmd := exec.Command("wsl", "--status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("WSL is not installed")
	}
	output := string(out)
	if strings.Contains(output, "WSL 2") || strings.Contains(output, "Default Version: 2") {
		fmt.Println("  ✓ WSL2 is installed and configured")
		return nil
	}
	if strings.Contains(output, "WSL 1") {
		fmt.Println("  ! WSL1 detected, upgrading to WSL2...")
		return upgradeToWSL2()
	}
	return fmt.Errorf("WSL2 not configured")
}

func InstallWSL() error {
	fmt.Println("  → Installing Windows Subsystem for Linux (WSL2)...")
	fmt.Println("    Running: wsl --install (this may take several minutes)")

	cmd := exec.Command("wsl", "--install", "--no-distribution")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wsl install: %w", err)
	}

	fmt.Println("  → Setting WSL2 as default...")
	_ = exec.Command("wsl", "--set-default-version", "2").Run()

	fmt.Println("  ✓ WSL2 installed. Restart required before Docker Desktop installation.")
	return nil
}

func upgradeToWSL2() error {
	cmd := exec.Command("wsl", "--set-default-version", "2")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
	}

	cmd := exec.Command(installerPath, "install", "--quiet", "--accept-license")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install Docker Desktop: %w", err)
	}

	fmt.Println("  ✓ Docker Desktop installed")
	fmt.Println("  ! A system restart may be required before Astra-System can start.")
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
