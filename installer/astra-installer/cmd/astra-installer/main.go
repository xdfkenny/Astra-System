package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/astra-service/astra-installer/internal/deploy"
	"github.com/astra-service/astra-installer/internal/prereq"
	"github.com/astra-service/astra-installer/internal/setup"
	"github.com/astra-service/astra-installer/internal/state"
)

var Version = "0.2.0"

type Config struct {
	InstallDir string
	DataDir    string
	Registry   string
	Tag        string
	KioskPort  string
	PostgresPW string
	GhcrToken  string
	Channel    string
	Silent     bool
	Version    bool
	Remove     bool
}

func main() {
	cfg := parseAndValidate()

	if cfg.Version {
		fmt.Printf("Astra-System Installer v%s\n", Version)
		os.Exit(0)
	}

	if cfg.Remove {
		doRemove(cfg)
		return
	}

	setupSignalHandler(cfg.DataDir)

	if cfg.Silent {
		doSilentInstall(cfg)
		return
	}

	doInteractiveInstall(cfg)
}

func parseAndValidate() Config {
	cfg := Config{
		KioskPort:  "80",
		PostgresPW: fmt.Sprintf("astra_%d", time.Now().Unix()),
		Channel:    "stable",
		Registry:   "ghcr.io/xdfkenny/astra-system",
		Tag:        "latest",
	}
	flag.StringVar(&cfg.InstallDir, "install-dir", defaultInstallDir(), "Install directory")
	flag.StringVar(&cfg.DataDir, "data-dir", defaultDataDir(), "Data directory")
	flag.StringVar(&cfg.Registry, "registry", cfg.Registry, "Docker registry")
	flag.StringVar(&cfg.Tag, "tag", cfg.Tag, "Docker image tag")
	flag.StringVar(&cfg.KioskPort, "kiosk-port", "80", "Kiosk host port")
	flag.StringVar(&cfg.PostgresPW, "db-password", cfg.PostgresPW, "PostgreSQL password")
	flag.StringVar(&cfg.GhcrToken, "ghcr-token", "", "GHCR token (read:packages)")
	flag.StringVar(&cfg.Channel, "channel", "stable", "Update channel (stable|beta|canary)")
	flag.BoolVar(&cfg.Silent, "silent", false, "Silent install (no prompts)")
	flag.BoolVar(&cfg.Remove, "remove", false, "Remove/stop Astra-System and clean up")
	flag.BoolVar(&cfg.Version, "version", false, "Print version")
	flag.Parse()

	if cfg.InstallDir != "" {
		abs, err := filepath.Abs(cfg.InstallDir)
		if err == nil {
			cfg.InstallDir = abs
		}
	}
	if cfg.DataDir != "" {
		abs, err := filepath.Abs(cfg.DataDir)
		if err == nil {
			cfg.DataDir = abs
		}
	}

	if port, err := strconv.Atoi(cfg.KioskPort); err != nil || port < 1 || port > 65535 {
		log.Fatalf("invalid --kiosk-port: %q (must be 1-65535)", cfg.KioskPort)
	}

	switch cfg.Channel {
	case "stable", "beta", "canary":
	default:
		log.Fatalf("invalid --channel: %q (must be stable, beta, or canary)", cfg.Channel)
	}

	if cfg.Registry == "" {
		log.Fatal("--registry must not be empty")
	}
	if cfg.Tag == "" {
		log.Fatal("--tag must not be empty")
	}

	return cfg
}

func defaultDataDir() string {
	if runtime.GOOS == "windows" {
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appdata, "Astra-System")
	}
	if runtime.GOOS == "darwin" {
		return "/usr/local/var/astra-system"
	}
	return "/var/lib/astra-system"
}

func defaultInstallDir() string {
	if runtime.GOOS == "windows" {
		progFiles := os.Getenv("PROGRAMFILES")
		if progFiles == "" {
			progFiles = filepath.Join(os.Getenv("SYSTEMDRIVE")+"\\", "Program Files")
		}
		return filepath.Join(progFiles, "Astra-System")
	}
	if runtime.GOOS == "darwin" {
		return "/Applications/Astra-System"
	}
	return "/opt/astra-system"
}

func setupSignalHandler(dataDir string) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\n\n  ╔══════════════════════════════════════════╗")
		fmt.Println("  ║  Installation cancelled by user         ║")
		fmt.Println("  ║  Re-run the installer to resume.       ║")
		fmt.Println("  ╚══════════════════════════════════════════╝")
		fmt.Println()
		os.Exit(1)
	}()
}

// ── Silent install (headless / CI) ─────────────────────────────

func doSilentInstall(cfg Config) {
	log.SetPrefix("[astra-install] ")

	if runtime.GOOS == "windows" {
		if err := prereq.CheckWSL(); err != nil {
			log.Fatalf("WSL check: %v", err)
		}
	}

	if err := prereq.CheckDocker(); err != nil {
		log.Fatalf("Docker check: %v", err)
	}
	if err := prereq.WaitForDocker(120 * time.Second); err != nil {
		log.Fatalf("Docker wait: %v", err)
	}
	if err := deploy.Run(deploy.Config{
		InstallDir: cfg.InstallDir,
		DataDir:    cfg.DataDir,
		Registry:   cfg.Registry,
		Tag:        cfg.Tag,
		KioskPort:  cfg.KioskPort,
		PostgresPW: cfg.PostgresPW,
		GhcrToken:  cfg.GhcrToken,
		Channel:    cfg.Channel,
	}); err != nil {
		log.Fatalf("deploy: %v", err)
	}
	if err := setup.Install(cfg.InstallDir, cfg.DataDir, cfg.Channel, true); err != nil {
		log.Printf("setup (non-fatal): %v", err)
	}
	fmt.Println("✓ Installation complete")
}

// ── Remove / Uninstall ────────────────────────────────────────

func doRemove(cfg Config) {
	fmt.Println()
	fmt.Println("  Removing Astra-System...")
	fmt.Println()

	if err := deploy.Remove(cfg.DataDir, cfg.InstallDir); err != nil {
		log.Printf("  ! deploy cleanup: %v", err)
	}

	if err := setup.Remove(cfg.DataDir); err != nil {
		log.Printf("  ! setup cleanup: %v", err)
	}

	fmt.Println("  ✓ Astra-System removed")
}

// ── Interactive install (state-machine wizard) ─────────────────

func doInteractiveInstall(cfg Config) {
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════╗")
	fmt.Printf("  ║     Astra-System Installer v%-12s ║\n", Version)
	fmt.Println("  ║  Production-grade Self-Checkout Platform ║")
	platform := runtime.GOOS + "/" + runtime.GOARCH
	fmt.Printf("  ║  Platform: %-27s ║\n", platform)
	fmt.Println("  ╚══════════════════════════════════════════╝")
	fmt.Println()

	dataDir := cfg.DataDir
	installDir := cfg.InstallDir

	s, err := state.Load(dataDir)
	if err != nil {
		log.Fatalf("state: %v", err)
	}

	fmt.Printf("  Install: %s\n", installDir)
	fmt.Printf("  Data:    %s\n", dataDir)
	fmt.Println()

	if runtime.GOOS != "windows" && s.Step == state.StepWSL {
		s.Step = state.StepDocker
		s.Save()
	}

	switch s.Step {
	case state.StepWSL:
		handleWSL(s)
	case state.StepDocker:
		handleDocker(s, dataDir)
	case state.StepDeploy:
		handleDeploy(s, installDir, dataDir, cfg)
	case state.StepDone:
		fmt.Println("  ✓ Astra-System is already installed and running.")
		fmt.Println()
		answer := prompt("  Reinstall? (y/N): ")
		if strings.EqualFold(answer, "y") {
			s.Step = state.StepWSL
			s.Save()
			handleWSL(s)
		}
	}
}

func stepCounter(current, total int) string {
	return fmt.Sprintf("  [%d/%d]", current, total)
}

func wizardStepCounts() (dockerStep, deployStep, total int) {
	if runtime.GOOS == "windows" {
		return 2, 3, 3
	}
	return 1, 2, 2
}

func handleWSL(s *state.State) {
	if runtime.GOOS != "windows" {
		s.Step = state.StepDocker
		s.Save()
		handleDocker(s, defaultDataDir())
		return
	}

	fmt.Println(stepCounter(1, 3) + " Checking WSL2...")
	fmt.Println()

	if err := prereq.CheckWSL(); err == nil {
		fmt.Println("  ✓ WSL2 is installed and running.")
		fmt.Println()
		s.Step = state.StepDocker
		s.Save()
		handleDocker(s, defaultDataDir())
		return
	}

	fmt.Println("  ! WSL2 is required for Docker Desktop.")
	fmt.Println("  ! The installer will run: wsl --install")
	fmt.Println("  ! This may take several minutes and requires a system restart.")
	fmt.Println()
	answer := prompt("  Install WSL2 now? (Y/n): ")
	if strings.EqualFold(answer, "n") {
		fmt.Println("\n  ✗ WSL2 is required. Please install it manually and re-run.")
		os.Exit(1)
	}

	if err := prereq.InstallWSL(); err != nil {
		fmt.Printf("\n  ✗ WSL installation failed: %v\n", err)
		fmt.Println("  Try running 'wsl --install' manually in an admin PowerShell.")
		os.Exit(1)
	}

	// If WSL2 is now working, continue without reboot
	if prereq.CheckWSL() == nil {
		fmt.Println("  ✓ WSL2 ready.")
		fmt.Println()
		s.Step = state.StepDocker
		s.Save()
		handleDocker(s, defaultDataDir())
		return
	}

	s.Step = state.StepDocker
	s.Save()

	fmt.Println()
	fmt.Println("  ═══════════════════════════════════════════")
	fmt.Println("  ✓ WSL2 installed. RESTART YOUR COMPUTER.")
	fmt.Println("  Run this installer again after restart.")
	fmt.Println("  ═══════════════════════════════════════════")
}

func handleDocker(s *state.State, dataDir string) {
	ds, _, tt := wizardStepCounts()
	fmt.Println(stepCounter(ds, tt) + " Checking Docker...")
	fmt.Println()

	if runtime.GOOS == "windows" {
		if err := prereq.CheckWSL(); err != nil {
			fmt.Println("  ! WSL2 is not ready yet.")
			fmt.Println("  ! Complete WSL2 setup first.")
			os.Exit(1)
		}
	}

	if err := prereq.CheckDocker(); err != nil {
		if !prereq.DockerInstalled() {
			fmt.Println("  ! Docker is not installed.")
			fmt.Println()
			answer := prompt("  Install Docker now? (Y/n): ")
			if strings.EqualFold(answer, "n") {
				fmt.Println("\n  ✗ Docker is required. Exiting.")
				os.Exit(1)
			}
			_ = prereq.InstallDockerDesktop(false)
			s.Step = state.StepDeploy
			s.Save()
			fmt.Println()
			fmt.Println("  ═══════════════════════════════════════════")
			fmt.Println("  ✓ Docker installed.")
			fmt.Println("  Start Docker, then re-run this installer.")
			fmt.Println("  ═══════════════════════════════════════════")
			return
		}
		fmt.Println("  ! Docker is installed but not running.")
		fmt.Println("  ! Waiting 30 seconds for Docker to start...")
		for i := 0; i < 30; i++ {
			time.Sleep(1 * time.Second)
			if err := prereq.CheckDocker(); err == nil {
				fmt.Println()
				fmt.Println("  ✓ Docker is now running.")
				fmt.Println()
				s.Step = state.StepDeploy
				if err := s.Save(); err != nil {
					log.Printf("warning: state save: %v", err)
				}
				handleDeploy(s, defaultInstallDir(), dataDir, Config{
					KioskPort: "80",
					Channel:   "stable",
					Registry:  "ghcr.io/xdfkenny/astra-system",
					Tag:       "latest",
				})
				return
			}
		}
		fmt.Println("  ! Docker still not running.")
		fmt.Println("  ! Start Docker Desktop, then re-run this installer.")
		os.Exit(1)
	}

	fmt.Println("  ✓ Docker is installed and running.")
	fmt.Println()
	s.Step = state.StepDeploy
	s.Save()
	handleDeploy(s, defaultInstallDir(), dataDir, Config{
		KioskPort: "80",
		Channel:   "stable",
		Registry:  "ghcr.io/xdfkenny/astra-system",
		Tag:       "latest",
	})
}

func handleDeploy(s *state.State, installDir, dataDir string, cfg Config) {
	_, ds, tt := wizardStepCounts()
	fmt.Println(stepCounter(ds, tt) + " Deploying Astra-System...")
	fmt.Println()

	if cfg.GhcrToken == "" {
		fmt.Println("  ! No GHCR token set. If images require auth, set --ghcr-token.")
	}

	err := deploy.Run(deploy.Config{
		InstallDir: installDir,
		DataDir:    dataDir,
		Registry:   cfg.Registry,
		Tag:        cfg.Tag,
		KioskPort:  cfg.KioskPort,
		PostgresPW: cfg.PostgresPW,
		GhcrToken:  cfg.GhcrToken,
		Channel:    cfg.Channel,
	})
	if err != nil {
		fmt.Printf("\n  ✗ Deployment failed: %v\n", err)
		fmt.Printf("  Check logs in: %s\n", filepath.Join(dataDir, "logs"))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("  → Registering update agent...")
	if rerr := setup.Install(installDir, dataDir, cfg.Channel, false); rerr != nil {
		fmt.Printf("  ! Note: update agent not registered: %v\n", rerr)
	}

	s.Step = state.StepDone
	if err := s.Save(); err != nil {
		log.Printf("warning: state save: %v", err)
	}

	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════╗")
	fmt.Println("  ║     Astra-System is now running!         ║")
	fmt.Println("  ║                                          ║")
	fmt.Printf("  ║     Kiosk:   http://localhost:%s          ║\n", cfg.KioskPort)
	fmt.Println("  ║     API:     http://localhost:8080       ║")
	fmt.Println("  ║                                          ║")
	fmt.Println("  ║  The update agent will keep your system  ║")
	fmt.Println("  ║  up to date automatically.               ║")
	fmt.Println("  ╚══════════════════════════════════════════╝")
}

func prompt(msg string) string {
	fmt.Print(msg)
	var input string
	fmt.Scanln(&input)
	return input
}
