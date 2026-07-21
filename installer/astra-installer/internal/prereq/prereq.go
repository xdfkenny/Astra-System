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
	output := string(out)

	if err != nil {
		if strings.Contains(output, "not installed") || strings.Contains(output, "not found") {
			return fmt.Errorf("WSL is not installed")
		}
		return fmt.Errorf("WSL check failed: %s", strings.TrimSpace(output))
	}

	if strings.Contains(output, "WSL 2") || strings.Contains(output, "Default Version: 2") {
		fmt.Println("  ✓ WSL2 is installed and configured")
		return nil
	}
	if strings.Contains(output, "WSL 1") {
		fmt.Println("  ! WSL1 detected, upgrading to WSL2...")
		return upgradeToWSL2()
	}
	if strings.Contains(output, "No installed") || strings.Contains(output, "no distribution") {
		return fmt.Errorf("WSL is installed but no Linux distribution is set up. Run 'wsl --install' manually or let the installer do it")
	}
	return fmt.Errorf("WSL is not properly configured: %s", strings.TrimSpace(output))
}

func InstallWSL() error {
	fmt.Println("  → Installing Windows Subsystem for Linux (WSL2)...")
	fmt.Println("    Running: wsl --install (this may take several minutes)")
	fmt.Println("    A Windows feature installation dialog may appear — follow the prompts.")

	cmd := exec.Command("wsl", "--install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wsl --install failed: %w", err)
	}

	fmt.Println("  → Setting WSL2 as default version...")
	setVer := exec.Command("wsl", "--set-default-version", "2")
	setVer.Stdout = os.Stdout
	setVer.Stderr = os.Stderr
	if err := setVer.Run(); err != nil {
		fmt.Printf("  ! Note: could not set WSL2 as default: %v\n", err)
		fmt.Println("  ! You can run 'wsl --set-default-version 2' manually after restart.")
	}

	fmt.Println("  ✓ WSL2 installed. A system restart is required.")
	fmt.Println("  ! After restart, run the Astra-System installer again.")
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
