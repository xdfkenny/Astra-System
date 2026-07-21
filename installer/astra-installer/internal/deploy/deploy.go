package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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
	GhcrToken  string
	Channel    string
}

// dockerCmd returns (binary, extraArgs) for docker compose.
func dockerCmd() (name string, args []string) {
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return "docker-compose", nil
	}
	return "docker", []string{"compose"}
}

func Run(cfg Config) error {
	if err := ensureDirectories(cfg.DataDir); err != nil {
		return fmt.Errorf("directories: %w", err)
	}

	if cfg.GhcrToken != "" {
		fmt.Println("  → Authenticating with container registry...")
		host := strings.TrimPrefix(cfg.Registry, "https://")
		host = strings.TrimPrefix(host, "http://")
		login := exec.Command("docker", "login", host, "-u", "xdfkenny", "--password-stdin")
		login.Stdin = strings.NewReader(cfg.GhcrToken)
		login.Stdout = os.Stdout
		login.Stderr = os.Stderr
		if err := login.Run(); err != nil {
			return fmt.Errorf("registry login: %w", err)
		}
		fmt.Println("  ✓ Logged in to container registry")
	}

	composeDir := filepath.Join(cfg.DataDir, "compose")
	if err := os.MkdirAll(composeDir, 0755); err != nil {
		return err
	}

	kioskImage := "kiosk-unified"
	if !imageExists(cfg.Registry, kioskImage, cfg.Tag) {
		kioskImage = "kiosk"
		fmt.Println("  ! kiosk-unified not available, falling back to kiosk")
	}

	fmt.Println("  → Generating docker-compose.yml...")
	if _, err := compose.Generate(compose.Config{
		Registry:   cfg.Registry,
		Tag:        cfg.Tag,
		KioskImage: kioskImage,
		KioskPort:  cfg.KioskPort,
		PostgresPW: cfg.PostgresPW,
		DataDir:    cfg.DataDir,
	}, composeDir); err != nil {
		return fmt.Errorf("generate compose: %w", err)
	}

	fmt.Println("  → Writing environment configuration...")
	envPath := filepath.Join(composeDir, ".env")
	envContent := fmt.Sprintf(`POSTGRES_USER=astra
POSTGRES_PASSWORD=%s
POSTGRES_DB=astra_service
KIOSK_PORT=%s
`, cfg.PostgresPW, cfg.KioskPort)
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("env file: %w", err)
	}

	confPath := filepath.Join(cfg.DataDir, "config", "astra.conf")
	confContent := fmt.Sprintf(`ASTRA_DATA_DIR=%s
ASTRA_UPDATE_CHANNEL=%s
ASTRA_COMPOSE_DIR=%s
ASTRA_LOG_DIR=%s
ASTRA_CURRENT_VERSION=%s
`, cfg.DataDir, cfg.Channel, composeDir, filepath.Join(cfg.DataDir, "logs"), cfg.Tag)
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		return fmt.Errorf("astra config: %w", err)
	}

	fmt.Println("  → Writing database initialization scripts...")
	if err := writeInitSQL(composeDir); err != nil {
		fmt.Printf("  ! Note: init scripts not written: %v\n", err)
	}

	dcName, dcArgs := dockerCmd()
	composeArgs := append(append([]string{}, dcArgs...),
		"-p", "astra-system", "-f", "docker-compose.yml")

	fmt.Println("  → Pulling Docker images (~2GB, this may take a while)...")
	if err := runCmd(composeDir, dcName, append(composeArgs, "pull", "--quiet")...); err != nil {
		return fmt.Errorf("pull images: %w", err)
	}

	fmt.Println("  → Starting services...")
	if err := runCmd(composeDir, dcName, append(composeArgs, "up", "-d")...); err != nil {
		return fmt.Errorf("start services: %w", err)
	}

	fmt.Println("  → Waiting for services to become healthy...")
	if err := waitHealthy(composeDir, dcName, composeArgs, 3*time.Minute); err != nil {
		fmt.Printf("  ! Note: %v\n", err)
		showUnhealthy(composeDir, dcName, composeArgs)
		fmt.Println("  ! Some services may still be starting. Check with:")
		fmt.Printf("  !   %s -p astra-system ps\n", strings.Join(append([]string{dcName}, dcArgs...), " "))
		return nil
	}

	return nil
}

func ensureDirectories(dataDir string) error {
	for _, d := range []string{
		dataDir,
		filepath.Join(dataDir, "compose"),
		filepath.Join(dataDir, "config"),
		filepath.Join(dataDir, "logs"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
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

func imageExists(registry, image, tag string) bool {
	ref := fmt.Sprintf("%s/%s:%s", registry, image, tag)
	if exec.Command("docker", "pull", ref).Run() == nil {
		return true
	}
	return exec.Command("docker", "manifest", "inspect", ref).Run() == nil
}

// waitHealthy polls docker compose ps until all services are healthy.
func waitHealthy(composeDir, dcName string, dcArgs []string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	psArgs := append(append([]string{}, dcArgs...), "ps", "--format", "{{.Name}}\t{{.Status}}")

	for time.Now().Before(deadline) {
		cmd := exec.Command(dcName, psArgs...)
		cmd.Dir = composeDir
		out, err := cmd.Output()
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		type svc struct{ name, status string }
		var running, starting, unhealthy int

		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) < 2 {
				continue
			}
			s := svc{name: parts[0], status: parts[1]}
			switch {
			case strings.Contains(s.status, "Exit"), strings.Contains(s.status, "unhealthy"):
				unhealthy++
			case strings.Contains(s.status, "Up"), strings.Contains(s.status, "healthy"):
				running++
			default:
				starting++
			}
		}

		if unhealthy == 0 && running > 0 && starting == 0 {
			return nil
		}

		fmt.Printf("\r  ─ Running: %d  Starting: %d  Unhealthy: %d  ", running, starting, unhealthy)
		time.Sleep(5 * time.Second)
	}
	fmt.Println()
	return fmt.Errorf("timeout after %v", timeout)
}

func showUnhealthy(composeDir, dcName string, dcArgs []string) {
	psArgs := append(append([]string{}, dcArgs...), "ps", "--format", "{{.Name}}\t{{.Status}}")
	out, err := exec.Command(dcName, psArgs...).Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Exit") || strings.Contains(line, "unhealthy") {
			fmt.Printf("  ! %s\n", line)
		}
	}
}

// ── Remove ─────────────────────────────────────────────────────

func Remove(dataDir, installDir string) error {
	composeDir := filepath.Join(dataDir, "compose")
	dcName, dcArgs := dockerCmd()
	composeArgs := append(append([]string{}, dcArgs...), "-p", "astra-system")

	if _, err := os.Stat(filepath.Join(composeDir, "docker-compose.yml")); err == nil {
		composeArgs = append(composeArgs, "-f", "docker-compose.yml")
		fmt.Println("  → Stopping containers...")
		down := exec.Command(dcName, append(composeArgs, "down", "--volumes")...)
		down.Dir = composeDir
		down.Stdout = os.Stdout
		down.Stderr = os.Stderr
		down.Run()
	}

	for _, d := range []string{dataDir, installDir} {
		if d == "" {
			continue
		}
		if _, err := os.Stat(d); err == nil {
			fmt.Printf("  → Removing %s...\n", d)
			if err := os.RemoveAll(d); err != nil {
				return fmt.Errorf("remove %s: %w", d, err)
			}
		}
	}
	return nil
}

// ── Cleanup ────────────────────────────────────────────────────

var cleanupFuncs []func()
var cleanupMu sync.Mutex

func RegisterCleanup(fn func()) {
	cleanupMu.Lock()
	cleanupFuncs = append(cleanupFuncs, fn)
	cleanupMu.Unlock()
}

func RunCleanup() {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	for i := len(cleanupFuncs) - 1; i >= 0; i-- {
		cleanupFuncs[i]()
	}
}
