package apply

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const StackName = "astra-system"

func UpdateDockerStack(dataDir, newTag string) error {
	composeDir := filepath.Join(dataDir, "compose")
	composeFile := filepath.Join(composeDir, "docker-compose.yml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found at %s", composeFile)
	}

	if err := backupCurrentConfig(dataDir); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	fmt.Println("  Pulling new images...")
	if err := pullImages(composeDir); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	fmt.Println("  Restarting services with new images...")
	if err := restartStack(composeDir); err != nil {
		return fmt.Errorf("restart: %w", err)
	}

	recordUpdate(dataDir, newTag)
	return nil
}

func pullImages(composeDir string) error {
	cmd := exec.Command("docker", "compose",
		"-p", StackName,
		"-f", filepath.Join(composeDir, "docker-compose.yml"),
		"pull", "--quiet",
	)
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func restartStack(composeDir string) error {
	cmd := exec.Command("docker", "compose",
		"-p", StackName,
		"-f", filepath.Join(composeDir, "docker-compose.yml"),
		"up", "-d", "--pull", "always",
	)
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopStack() error {
	composeDir := findComposeDir()
	if composeDir == "" {
		return nil
	}
	cmd := exec.Command("docker", "compose",
		"-p", StackName,
		"-f", filepath.Join(composeDir, "docker-compose.yml"),
		"down", "--timeout", "30",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findComposeDir() string {
	candidates := []string{
		filepath.Join(os.Getenv("PROGRAMDATA"), "Astra-System", "compose"),
		filepath.Join(os.Getenv("ProgramFiles"), "Astra-System", "compose"),
	}
	for _, d := range candidates {
		if _, err := os.Stat(filepath.Join(d, "docker-compose.yml")); err == nil {
			return d
		}
	}
	return ""
}

func backupCurrentConfig(dataDir string) error {
	backupDir := filepath.Join(dataDir, "backups",
		fmt.Sprintf("pre-update-%s", time.Now().UTC().Format("20060102T150405Z")))

	composeDir := filepath.Join(dataDir, "compose")
	if _, err := os.Stat(composeDir); os.IsNotExist(err) {
		return nil
	}

	return copyDir(composeDir, backupDir)
}

func recordUpdate(dataDir, version string) {
	recordDir := filepath.Join(dataDir, "updates")
	os.MkdirAll(recordDir, 0755)
	recordFile := filepath.Join(recordDir,
		fmt.Sprintf("update-%s-%s.txt", time.Now().UTC().Format("20060102T150405Z"), sanitize(version)))
	os.WriteFile(recordFile, []byte(version+"\n"), 0644)
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			return r
		}
		return '_'
	}, s)
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
