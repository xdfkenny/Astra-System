package prereq

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CheckWSL() bool {
	out, err := exec.Command("wsl", "--status").CombinedOutput()
	if err != nil {
		return false
	}
	output := string(out)
	return strings.Contains(output, "WSL 2") || strings.Contains(output, "Default Version: 2")
}

func InstallWSL() error {
	fmt.Println("  Running: wsl --install")
	fmt.Println("  A Windows Features dialog may appear — follow the prompts.")
	fmt.Println()

	cmd := exec.Command("wsl", "--install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wsl --install: %w", err)
	}

	fmt.Println()
	_ = exec.Command("wsl", "--set-default-version", "2").Run()
	return nil
}

func DockerInstalled() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	_, err := exec.Command("docker", "info", "--format", "{{.ServerVersion}}").Output()
	return err == nil
}

func CheckDocker() bool {
	if !DockerInstalled() {
		return false
	}
	fmt.Println("  ✓ Docker Desktop is running")
	return true
}

func InstallDockerDesktop() error {
	url := "https://desktop.docker.com/win/stable/Docker%20Desktop%20Installer.exe"
	path := filepath.Join(os.TempDir(), "DockerDesktopInstaller.exe")

	fmt.Println("  Downloading Docker Desktop...")
	if err := download(url, path); err != nil {
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

func download(url, dest string) error {
	cmd := exec.Command("powershell",
		"-Command",
		fmt.Sprintf("(New-Object System.Net.WebClient).DownloadFile('%s','%s')", url, dest),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
