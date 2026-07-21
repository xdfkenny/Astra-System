package deploy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/astra-service/astra-installer/internal/compose"
)

const StackName = "astra-system"

type Config struct {
	InstallDir string
	DataDir    string
	Registry   string
	Tag        string
	KioskPort  string
	PostgresPW string
	Silent     bool
}

func Run(cfg Config) error {
	if err := ensureDirectories(cfg.DataDir); err != nil {
		return fmt.Errorf("directories: %w", err)
	}

	if !cfg.Silent {
		fmt.Println("→ Generating Docker Compose configuration...")
	}

	composeDir := filepath.Join(cfg.DataDir, "compose")
	if err := os.MkdirAll(composeDir, 0755); err != nil {
		return err
	}

	kioskImage := "kiosk-unified"
	if !imageExists(cfg.Registry, kioskImage, cfg.Tag) {
		kioskImage = "kiosk"
		if cfg.Silent {
			fmt.Println("  ! kiosk-unified not available, falling back to kiosk")
		}
	}

	_, err := compose.Generate(compose.Config{
		Registry:   cfg.Registry,
		Tag:        cfg.Tag,
		KioskImage: kioskImage,
		DataDir:    cfg.DataDir,
		KioskPort:  cfg.KioskPort,
		PostgresPW: cfg.PostgresPW,
	}, composeDir)
	if err != nil {
		return fmt.Errorf("generate compose: %w", err)
	}

	if !cfg.Silent {
		fmt.Println("  ✓ docker-compose.yml generated")
	}

	if err := writeEnvFile(cfg.DataDir); err != nil {
		return fmt.Errorf("env file: %w", err)
	}

	if !cfg.Silent {
		fmt.Println("  ✓ Environment configured")
	}

	if !cfg.Silent {
		fmt.Println("→ Preparing database initialization scripts...")
	}
	if err := writeInitSQL(composeDir); err != nil {
		return fmt.Errorf("init sql: %w", err)
	}

	if !cfg.Silent {
		fmt.Println("→ Pulling Docker images...")
	}
	if err := dockerComposePull(composeDir); err != nil {
		return fmt.Errorf("pull images: %w", err)
	}
	if !cfg.Silent {
		fmt.Println("  ✓ Images downloaded")
	}

	if !cfg.Silent {
		fmt.Println("→ Starting services...")
	}
	if err := dockerComposeUp(composeDir); err != nil {
		return fmt.Errorf("start services: %w", err)
	}

	if !cfg.Silent {
		fmt.Println("  ✓ Services started")
	}

	if !cfg.Silent {
		fmt.Println("→ Waiting for services to become healthy...")
	}
	if err := waitForServices(composeDir, 120*time.Second); err != nil {
		fmt.Printf("  ! Some services may not be ready: %v\n", err)
	}

	return nil
}

func ensureDirectories(dataDir string) error {
	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "compose"),
		filepath.Join(dataDir, "config"),
		filepath.Join(dataDir, "logs"),
		filepath.Join(dataDir, "updates"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

func writeEnvFile(dataDir string) error {
	envPath := filepath.Join(dataDir, "compose", ".env")
	content := `POSTGRES_USER=astra
POSTGRES_DB=astra_service
KIOSK_PORT=80
`
	return os.WriteFile(envPath, []byte(content), 0644)
}

func dockerComposePull(composeDir string) error {
	cmd := exec.Command("docker", "compose",
		"-p", StackName,
		"-f", filepath.Join(composeDir, "docker-compose.yml"),
		"pull",
		"--quiet",
	)
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dockerComposeUp(composeDir string) error {
	cmd := exec.Command("docker", "compose",
		"-p", StackName,
		"-f", filepath.Join(composeDir, "docker-compose.yml"),
		"up", "-d",
	)
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForServices(composeDir string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		healthy, err := allServicesHealthy(composeDir)
		if err != nil {
			return err
		}
		if healthy {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	cmd := exec.Command("docker", "compose",
		"-p", StackName,
		"-f", filepath.Join(composeDir, "docker-compose.yml"),
		"ps", "--format", "table {{.Name}}\t{{.Status}}",
	)
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return fmt.Errorf("timeout waiting for services after %v", timeout)
}

func allServicesHealthy(composeDir string) (bool, error) {
	cmd := exec.Command("docker", "compose",
		"-p", StackName,
		"-f", filepath.Join(composeDir, "docker-compose.yml"),
		"ps", "--format", "{{.Name}}\t{{.Status}}",
	)
	cmd.Dir = composeDir
	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return false, nil
	}

	for _, line := range lines {
		if strings.Contains(line, "Exit") || strings.Contains(line, "unhealthy") {
			return false, nil
		}
		if !strings.Contains(line, "Up") && !strings.Contains(line, "healthy") {
			return false, nil
		}
	}

	return true, nil
}

func ReadLine(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func imageExists(registry, image, tag string) bool {
	fullImage := fmt.Sprintf("%s/%s:%s", registry, image, tag)
	cmd := exec.Command("docker", "manifest", "inspect", fullImage)
	return cmd.Run() == nil
}
