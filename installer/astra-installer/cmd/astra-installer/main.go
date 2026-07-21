package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/astra-service/astra-installer/internal/prereq"
	"github.com/astra-service/astra-installer/internal/setup"
)

var Version = "0.2.0"

type Config struct {
	InstallDir string
	DataDir    string
	Channel    string
	Silent     bool
	NoDocker   bool
	Version    bool
}

func main() {
	cfg := parseFlags()

	if cfg.Version {
		fmt.Printf("Astra-System Installer v%s\n", Version)
		os.Exit(0)
	}

	log.SetPrefix("[astra-install] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	if !cfg.Silent {
		printBanner()
	}

	if err := run(cfg); err != nil {
		log.Fatalf("installation failed: %v", err)
	}

	if !cfg.Silent {
		fmt.Println("\n✓ Astra-System installed successfully.")
		fmt.Printf("  Install path: %s\n", cfg.InstallDir)
		fmt.Printf("  Data path:    %s\n", cfg.DataDir)
		fmt.Printf("  Channel:      %s\n", cfg.Channel)
		fmt.Println("\n  The update agent will keep your system up to date.")
		fmt.Println("  Open http://localhost to access the kiosk.")
	}
}

func parseFlags() Config {
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		programData = filepath.Join(os.Getenv("SYSTEMDRIVE")+"\\", "ProgramData")
	}
	defaultDataDir := filepath.Join(programData, "Astra-System")

	programFiles := os.Getenv("PROGRAMFILES")
	if programFiles == "" {
		programFiles = filepath.Join(os.Getenv("SYSTEMDRIVE")+"\\", "Program Files")
	}
	defaultInstallDir := filepath.Join(programFiles, "Astra-System")

	cfg := Config{}
	flag.StringVar(&cfg.InstallDir, "install-dir", defaultInstallDir, "Application installation directory")
	flag.StringVar(&cfg.DataDir, "data-dir", defaultDataDir, "Application data directory")
	flag.StringVar(&cfg.Channel, "channel", "stable", "Update channel (stable, beta, canary)")
	flag.BoolVar(&cfg.Silent, "silent", false, "Silent installation (no prompts)")
	flag.BoolVar(&cfg.NoDocker, "no-docker", false, "Skip Docker Desktop check")
	flag.BoolVar(&cfg.Version, "version", false, "Print version and exit")
	flag.Parse()
	return cfg
}

func printBanner() {
	fmt.Println(`
   ╔══════════════════════════════════════════╗
   ║        Astra-System Installer v` + Version + `        ║
   ║  Production-grade Self-Checkout Platform ║
   ╚══════════════════════════════════════════╝
	`)
}

func run(cfg Config) error {
	if !cfg.NoDocker {
		if !cfg.Silent {
			fmt.Println("→ Checking prerequisites...")
		}
		status := prereq.CheckDocker()
		switch status {
		case prereq.DockerNotFound:
			return fmt.Errorf("Docker Desktop not found. Install Docker Desktop from https://docs.docker.com/desktop/setup/install/windows-install/ and try again, or use --no-docker to skip")
		case prereq.DockerNotRunning:
			return fmt.Errorf("Docker Desktop is installed but not running. Start Docker Desktop and try again")
		case prereq.DockerOK:
			if !cfg.Silent {
				fmt.Println("  ✓ Docker Desktop is installed and running")
			}
		}
	}

	if !cfg.Silent {
		fmt.Println("→ Setting up Astra-System...")
	}

	if err := setup.Install(cfg.InstallDir, cfg.DataDir, cfg.Channel, cfg.Silent); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	return nil
}
