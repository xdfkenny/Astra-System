package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/astra-service/astra-updater/internal/apply"
	"github.com/astra-service/astra-updater/internal/check"
	"github.com/astra-service/astra-updater/internal/download"
)

const (
	ServiceName = "AstraUpdateAgent"
	Version     = "0.2.0"
)

var (
	installDir string
	dataDir    string
	channel    string
	interval   time.Duration
)

type Config struct {
	InstallDir string `json:"install_dir"`
	DataDir    string `json:"data_dir"`
	Channel    string `json:"channel"`
	Interval   string `json:"interval"`
	CurrentVer string `json:"current_version"`
}

func main() {
	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	installCmd.StringVar(&dataDir, "data-dir", defaultDataDir(), "Data directory")
	installCmd.StringVar(&installDir, "install-dir", defaultInstallDir(), "Install directory")

	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runCmd.StringVar(&dataDir, "data-dir", defaultDataDir(), "Data directory")
	runCmd.StringVar(&installDir, "install-dir", defaultInstallDir(), "Install directory")
	runCmd.StringVar(&channel, "channel", "stable", "Update channel")
	runCmd.DurationVar(&interval, "interval", 6*time.Hour, "Check interval")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "install":
		installCmd.Parse(os.Args[2:])
		doInstall()
	case "remove":
		removeCmd.Parse(os.Args[2:])
		doRemove()
	case "run":
		runCmd.Parse(os.Args[2:])
		doRun()
	case "version":
		fmt.Printf("Astra-Update-Agent v%s\n", Version)
		os.Exit(0)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`Astra Update Agent v%s — Windows Service for auto-updating Astra-System

Usage:
  astra-updater install  [--data-dir <path>] [--install-dir <path>]
  astra-updater remove
  astra-updater run      [--data-dir <path>] [--install-dir <path>] [--channel <name>] [--interval <duration>]
  astra-updater version
`, Version)
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

func doInstall() {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatalf("connect to service manager: %v", err)
	}
	defer m.Disconnect()

	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("get executable path: %v", err)
	}

	svcConfig := mgr.Config{
		ServiceType:  windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:    windows.SERVICE_AUTO_START,
		ErrorControl: windows.SERVICE_ERROR_NORMAL,
		DisplayName:  "Astra-System Update Agent",
		Description:  "Checks for Astra-System updates via GitHub Releases and applies them automatically.",
	}

	s, err := m.CreateService(
		ServiceName,
		exe,
		mgr.Config(svcConfig),
		"run",
		"--data-dir", dataDir,
		"--install-dir", installDir,
	)
	if err != nil {
		log.Fatalf("create service: %v", err)
	}
	defer s.Close()

	fmt.Println("✓ AstraUpdateAgent service installed successfully")
}

func doRemove() {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatalf("connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		log.Fatalf("open service: %v", err)
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	if err != nil {
		log.Printf("note: service stop: %v", err)
	}

	if err := s.Delete(); err != nil {
		log.Fatalf("delete service: %v", err)
	}

	fmt.Println("✓ AstraUpdateAgent service removed successfully")
}

func doRun() {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("detect service context: %v", err)
	}

	if isSvc {
		if err := svc.Run(ServiceName, &handler{}); err != nil {
			log.Fatalf("service run: %v", err)
		}
	} else {
		fmt.Println("Running in foreground mode. Press Ctrl+C to stop.")
		runForeground()
	}
}

func runForeground() {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	fmt.Println("  Channel:", channel)
	fmt.Println("  Interval:", interval)
	fmt.Println("  Data dir:", dataDir)
	fmt.Println()

	checkAndUpdate()

	for {
		select {
		case <-ticker.C:
			checkAndUpdate()
		case <-sig:
			fmt.Println("\nShutting down.")
			return
		}
	}
}

func checkAndUpdate() {
	log.Println("Checking for updates...")

	cfg := loadConfig()

	release, err := check.LatestRelease(channel, cfg.CurrentVer)
	if err != nil {
		log.Printf("check update: %v", err)
		return
	}
	if release == nil {
		log.Println("Already up to date.")
		return
	}

	log.Printf("Update available: %s", release.TagName)

	assetPath, checksum, err := download.Asset(release, dataDir)
	if err != nil {
		log.Printf("download: %v", err)
		return
	}

	if err := apply.Update(assetPath, checksum, installDir, dataDir); err != nil {
		log.Printf("apply: %v", err)
		return
	}

	saveConfig(release.TagName)
	log.Printf("Update to %s applied successfully", release.TagName)
}

func loadConfig() Config {
	cfgPath := filepath.Join(dataDir, "config", "astra.conf")
	cfg := Config{
		Channel:    channel,
		Interval:   interval.String(),
		CurrentVer: "0.0.0",
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ASTRA_UPDATE_CHANNEL=") {
			cfg.Channel = strings.TrimPrefix(line, "ASTRA_UPDATE_CHANNEL=")
		}
		if strings.HasPrefix(line, "ASTRA_INSTALL_DIR=") {
			cfg.InstallDir = strings.TrimPrefix(line, "ASTRA_INSTALL_DIR=")
		}
		if strings.HasPrefix(line, "ASTRA_DATA_DIR=") {
			cfg.DataDir = strings.TrimPrefix(line, "ASTRA_DATA_DIR=")
		}
	}

	return cfg
}

func saveConfig(version string) {
	cfgPath := filepath.Join(dataDir, "config", "astra.conf")
	content := fmt.Sprintf(`{"current_version":"%s","last_update":"%s"}`, version, time.Now().UTC().Format(time.RFC3339))
	_ = os.WriteFile(cfgPath, []byte(content), 0644)
}

type handler struct{}

func (h *handler) Execute(args []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	checkAndUpdate()

	for {
		select {
		case <-ticker.C:
			checkAndUpdate()
		case c := <-requests:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				return false, 0
			}
		}
	}
}
