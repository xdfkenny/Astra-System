package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/astra-service/astra-installer/internal/compose"
)

type Config struct {
	InstallDir string
	DataDir    string
	Registry   string
	Tag        string
	KioskPort  string
	PostgresPW string
}

func Run(cfg Config) error {
	composeDir := filepath.Join(cfg.DataDir, "compose")
	if err := os.MkdirAll(composeDir, 0755); err != nil {
		return err
	}

	kioskImage := "kiosk"
	if dockerManifestExists(cfg.Registry, "kiosk-unified", cfg.Tag) {
		kioskImage = "kiosk-unified"
	}

	fmt.Println("  → Generating docker-compose.yml...")
	_, err := compose.Generate(compose.Config{
		Registry:   cfg.Registry,
		Tag:        cfg.Tag,
		KioskImage: kioskImage,
		KioskPort:  cfg.KioskPort,
		PostgresPW: cfg.PostgresPW,
		DataDir:    cfg.DataDir,
	}, composeDir)
	if err != nil {
		return fmt.Errorf("generate compose: %w", err)
	}

	fmt.Println("  → Writing database initialization scripts...")
	if err := writeInitSQL(composeDir); err != nil {
		return fmt.Errorf("init sql: %w", err)
	}

	fmt.Println("  → Pulling Docker images (~2GB, this may take a while)...")
	if err := runCmd(composeDir, "docker", "compose", "-p", "astra-system", "-f", "docker-compose.yml", "pull", "--quiet"); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	fmt.Println("  → Starting services...")
	if err := runCmd(composeDir, "docker", "compose", "-p", "astra-system", "-f", "docker-compose.yml", "up", "-d"); err != nil {
		return fmt.Errorf("up: %w", err)
	}

	fmt.Println("  → Waiting for services to become healthy...")
	if err := waitHealthy(composeDir, 2*time.Minute); err != nil {
		fmt.Printf("  ! Note: %v\n", err)
		fmt.Println("  ! Some services may still be starting.")
	}

	return nil
}

func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dockerManifestExists(registry, image, tag string) bool {
	ref := fmt.Sprintf("%s/%s:%s", registry, image, tag)
	return exec.Command("docker", "manifest", "inspect", ref).Run() == nil
}

func waitHealthy(composeDir string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command("docker", "compose", "-p", "astra-system", "-f", "docker-compose.yml", "ps", "--format", "{{.Name}}\t{{.Status}}")
		cmd.Dir = composeDir
		out, err := cmd.Output()
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		allUp := true
		hasServices := false
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			hasServices = true
			if strings.Contains(line, "Exit") || strings.Contains(line, "unhealthy") {
				allUp = false
			}
		}

		if hasServices && allUp {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("timeout waiting for healthy services")
}
