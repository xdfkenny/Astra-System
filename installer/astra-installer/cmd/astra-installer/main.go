package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/astra-service/astra-installer/internal/deploy"
	"github.com/astra-service/astra-installer/internal/prereq"
	"github.com/astra-service/astra-installer/internal/state"
)

var Version = "0.2.0"

func main() {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════╗")
	fmt.Printf("  ║     Astra-System Installer v%s        ║\n", Version)
	fmt.Println("  ║  Production-grade Self-Checkout Platform ║")
	fmt.Println("  ╚══════════════════════════════════════════╝")
	fmt.Println()

	dataDir := defaultDataDir()
	installDir := defaultInstallDir()

	s, err := state.Load(dataDir)
	if err != nil {
		log.Fatalf("state: %v", err)
	}

	fmt.Printf("  Install: %s\n", installDir)
	fmt.Printf("  Data:    %s\n", dataDir)
	fmt.Println()

	switch s.Step {
	case state.StepWSL:
		handleWSL(s)
	case state.StepDocker:
		handleDocker(s)
	case state.StepDeploy:
		handleDeploy(s, installDir, dataDir)
	default:
		handleDeploy(s, installDir, dataDir)
	}
}

func handleWSL(s *state.State) {
	fmt.Println("  [1/3] Checking WSL2...")
	fmt.Println()

	if prereq.CheckWSL() {
		fmt.Println()
		fmt.Println("  ✓ WSL2 is installed and running.")
		fmt.Println()
		s.Step = state.StepDocker
		s.Save()
		handleDocker(s)
		return
	}

	fmt.Println("  ! WSL2 is required for Docker Desktop.")
	fmt.Println("  ! The installer will run: wsl --install")
	fmt.Println("  ! This may take several minutes and requires a system restart.")
	fmt.Println()
	answer := prompt("  Install WSL2 now? (Y/n): ")
	if answer == "n" || answer == "N" {
		fmt.Println("\n  ✗ WSL2 is required. Please install it manually and re-run.")
		os.Exit(1)
	}

	if err := prereq.InstallWSL(); err != nil {
		fmt.Printf("\n  ✗ WSL installation failed: %v\n", err)
		fmt.Println("  Try running 'wsl --install' manually in an admin PowerShell.")
		os.Exit(1)
	}

	s.Step = state.StepDocker
	s.Save()

	fmt.Println()
	fmt.Println("  ═══════════════════════════════════════════")
	fmt.Println("  ✓ WSL2 installed. RESTART YOUR COMPUTER.")
	fmt.Println("  Run this installer again after restart.")
	fmt.Println("  ═══════════════════════════════════════════")
}

func handleDocker(s *state.State) {
	fmt.Println("  [2/3] Checking Docker Desktop...")
	fmt.Println()

	if prereq.CheckWSL() && prereq.CheckDocker() {
		fmt.Println()
		fmt.Println("  ✓ Docker Desktop is installed and running.")
		fmt.Println()
		s.Step = state.StepDeploy
		s.Save()
		handleDeploy(s, defaultInstallDir(), defaultDataDir())
		return
	}

	if !prereq.DockerInstalled() {
		fmt.Println("  ! Docker Desktop is not installed.")
		fmt.Println("  ! The installer will download and install it.")
		fmt.Println()
		answer := prompt("  Install Docker Desktop now? (Y/n): ")
		if answer == "n" || answer == "N" {
			fmt.Println("\n  ✗ Docker Desktop is required. Install it manually and re-run.")
			os.Exit(1)
		}

		if err := prereq.InstallDockerDesktop(); err != nil {
			fmt.Printf("\n  ✗ Docker installation failed: %v\n", err)
			fmt.Println("  Download from: https://docs.docker.com/desktop/setup/install/windows-install/")
			os.Exit(1)
		}

		s.Step = state.StepDeploy
		s.Save()

		fmt.Println()
		fmt.Println("  ═══════════════════════════════════════════")
		fmt.Println("  ✓ Docker Desktop installed.")
		fmt.Println("  Start Docker Desktop, then re-run this installer.")
		fmt.Println("  ═══════════════════════════════════════════")
		return
	}

	fmt.Printf("  ! Docker is installed but the daemon is not running.\n")
	fmt.Println("  ! Start Docker Desktop and re-run this installer.")
	os.Exit(1)
}

func handleDeploy(s *state.State, installDir, dataDir string) {
	fmt.Println("  [3/3] Deploying Astra-System...")
	fmt.Println()

	if err := deploy.Run(deploy.Config{
		InstallDir: installDir,
		DataDir:    dataDir,
		Registry:   "ghcr.io/xdfkenny/astra-system",
		Tag:        "latest",
		KioskPort:  "80",
		PostgresPW: "astra-system",
	}); err != nil {
		fmt.Printf("\n  ✗ Deployment failed: %v\n", err)
		fmt.Printf("  Check logs in: %s\n", filepath.Join(dataDir, "logs"))
		os.Exit(1)
	}

	s.Step = state.StepDone
	s.Save()

	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════╗")
	fmt.Println("  ║     Astra-System is now running!         ║")
	fmt.Println("  ║                                          ║")
	fmt.Println("  ║     Kiosk:   http://localhost             ║")
	fmt.Println("  ║     API:     http://localhost:8080         ║")
	fmt.Println("  ║                                          ║")
	fmt.Println("  ║  The update agent will keep your system  ║")
	fmt.Println("  ║  up to date automatically.               ║")
	fmt.Println("  ╚══════════════════════════════════════════╝")
}

func defaultDataDir() string {
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		programData = filepath.Join(os.Getenv("SYSTEMDRIVE")+"\\", "ProgramData")
	}
	return filepath.Join(programData, "Astra-System")
}

func defaultInstallDir() string {
	programFiles := os.Getenv("PROGRAMFILES")
	if programFiles == "" {
		programFiles = filepath.Join(os.Getenv("SYSTEMDRIVE")+"\\", "Program Files")
	}
	return filepath.Join(programFiles, "Astra-System")
}

func prompt(msg string) string {
	fmt.Print(msg)
	var input string
	fmt.Scanln(&input)
	return input
}
