//go:build windows

package prereq

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CheckWSL() error {
	out, err := exec.Command("powershell", "-Command", "wsl --status").Output()
	if err != nil {
		return fmt.Errorf("WSL not available: %w", err)
	}
	output := strings.ToLower(string(out))
	if strings.Contains(output, "default version: 2") || strings.Contains(output, "wsl 2") {
		return nil
	}
	return fmt.Errorf("WSL 2 is not the default version")
}

func wslDistros() []string {
	out, err := exec.Command("powershell", "-Command", "wsl --list --quiet").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

func InstallWSL() error {
	distros := wslDistros()
	if len(distros) > 0 {
		fmt.Println("  WSL has existing distributions.")
		fmt.Println("  Updating WSL and setting version 2 as default...")
		_ = exec.Command("wsl", "--update").Run()
		if err := exec.Command("wsl", "--set-default-version", "2").Run(); err != nil {
			fmt.Printf("  ! Could not set WSL2 as default: %v\n", err)
			fmt.Println("  ! Run this in admin PowerShell: wsl --set-default-version 2")
		}
		fmt.Println("  ✓ WSL configured. Check Docker Desktop now supports WSL2 backend.")
		return nil
	}

	fmt.Println("  Running: wsl --install")
	fmt.Println("  A Windows Features dialog may appear — follow the prompts.")
	fmt.Println()

	cmd := exec.Command("wsl", "--install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wsl --install failed: %w (try running it manually in admin PowerShell)", err)
	}

	fmt.Println()
	_ = exec.Command("wsl", "--set-default-version", "2").Run()
	return nil
}

func CheckDocker() error {
	if !DockerInstalled() {
		return fmt.Errorf("Docker not found on PATH")
	}
	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker daemon not running: %w", err)
	}
	fmt.Println("  ✓ Docker Desktop is running")
	return nil
}

func InstallDockerDesktop(silent bool) error {
	url := "https://desktop.docker.com/win/stable/Docker%20Desktop%20Installer.exe"
	path := filepath.Join(os.TempDir(), "DockerDesktopInstaller.exe")

	fmt.Println("  Downloading Docker Desktop...")
	if err := downloadFile(url, path); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	fmt.Println("  Installing Docker Desktop (this may take several minutes)...")
	cmd := exec.Command(path, "install", "--quiet", "--accept-license")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	return nil
}

func downloadFile(url, dest string) error {
	cmd := exec.Command("powershell",
		"-Command",
		fmt.Sprintf("(New-Object System.Net.WebClient).DownloadFile('%s','%s')", url, dest),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
