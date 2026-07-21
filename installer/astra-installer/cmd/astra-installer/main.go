package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/astra-service/astra-installer/internal/deploy"
	"github.com/astra-service/astra-installer/internal/prereq"
)

var Version = "0.2.0"

type Config struct {
	InstallDir string
	DataDir    string
	Registry   string
	Tag        string
	KioskPort  string
	PostgresPW string
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

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  Astra-System is now running!")
	fmt.Println()
	fmt.Printf("  Kiosk:    http://localhost:%s\n", cfg.KioskPort)
	fmt.Println("  Dashboard: http://localhost:8080")
	fmt.Println()
	fmt.Println("  The update agent will keep your system")
	fmt.Println("  up to date automatically.")
	fmt.Println("═══════════════════════════════════════════")
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

	cfg := Config{
		KioskPort:  "80",
		PostgresPW: fmt.Sprintf("astra_%d", time.Now().Unix()),
	}
	flag.StringVar(&cfg.InstallDir, "install-dir", defaultInstallDir, "Application installation directory")
	flag.StringVar(&cfg.DataDir, "data-dir", defaultDataDir, "Application data directory")
	flag.StringVar(&cfg.Registry, "registry", "ghcr.io/xdfkenny/astra-system", "Docker image registry")
	flag.StringVar(&cfg.Tag, "tag", "latest", "Docker image tag")
	flag.StringVar(&cfg.KioskPort, "kiosk-port", "80", "Host port for the kiosk web UI")
	flag.StringVar(&cfg.PostgresPW, "db-password", cfg.PostgresPW, "PostgreSQL password (auto-generated)")
	flag.BoolVar(&cfg.Silent, "silent", false, "Silent installation (no prompts)")
	flag.BoolVar(&cfg.NoDocker, "no-docker", false, "Skip Docker Desktop check and install")
	flag.BoolVar(&cfg.Version, "version", false, "Print version and exit")
	flag.Parse()
	return cfg
}

func printBanner() {
	fmt.Println(`
  ╔═══════════════════════════════════════════╗
  ║        Astra-System Installer v` + Version + `        ║
  ║  Production-grade Self-Checkout Platform  ║
  ╚═══════════════════════════════════════════╝
	`)
}

func run(cfg Config) error {
	if !cfg.NoDocker {
		if !cfg.Silent {
			fmt.Println("→ Checking prerequisites...")
		}

		if err := prereq.CheckWSL(); err != nil {
			fmt.Printf("\n  ! %v\n", err)
			answer := "y"
			if !cfg.Silent {
				answer = deploy.ReadLine("  Install WSL2 now? (Y/n): ")
			}
			if answer != "n" && answer != "N" {
				if err := prereq.InstallWSL(); err != nil {
					return fmt.Errorf("install WSL: %w", err)
				}
				fmt.Println("\n  ✓ WSL2 installed")
				fmt.Println("  ! Please restart your computer and run the installer again")
				fmt.Println("  ! to continue with Docker Desktop setup.")
				os.Exit(0)
			}
			return fmt.Errorf("WSL2 is required for Docker Desktop")
		}

		if err := prereq.CheckDocker(); err != nil {
			fmt.Printf("\n  ! %v\n", err)
			answer := "y"
			if !cfg.Silent {
				answer = deploy.ReadLine("  Install Docker Desktop now? (Y/n): ")
			}
			if answer != "n" && answer != "N" {
				if err := prereq.InstallDockerDesktop(cfg.Silent); err != nil {
					return fmt.Errorf("install Docker Desktop: %w", err)
				}
				fmt.Println("  ! Please restart your computer and run the installer again")
				fmt.Println("  ! to complete the Astra-System setup.")
				os.Exit(0)
			}
			return fmt.Errorf("Docker Desktop is required")
		}
	}

	if !cfg.Silent {
		fmt.Println()
		fmt.Println("→ Deploying Astra-System...")
		fmt.Printf("  Images:  %s/%s:%s\n", cfg.Registry, "{services}", cfg.Tag)
		fmt.Printf("  Kiosk:   http://localhost:%s\n", cfg.KioskPort)
		fmt.Printf("  Data:    %s\n", cfg.DataDir)
		fmt.Println()
	}

	if err := deploy.Run(deploy.Config{
		InstallDir: cfg.InstallDir,
		DataDir:    cfg.DataDir,
		Registry:   cfg.Registry,
		Tag:        cfg.Tag,
		KioskPort:  cfg.KioskPort,
		PostgresPW: cfg.PostgresPW,
		Silent:     cfg.Silent,
	}); err != nil {
		return err
	}

	if !cfg.Silent {
		fmt.Println("\n  ✓ All services deployed successfully!")
	}

	return nil
}
