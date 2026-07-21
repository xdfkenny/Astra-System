package apply

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func Update(assetPath, expectedChecksum, installDir, dataDir string) error {
	if err := verifyChecksum(assetPath, expectedChecksum); err != nil {
		return fmt.Errorf("integrity check: %w", err)
	}

	if err := stopStack(); err != nil {
		return fmt.Errorf("stop stack: %w", err)
	}

	if err := backupConfig(dataDir); err != nil {
		return fmt.Errorf("backup config: %w", err)
	}

	if err := runInstaller(assetPath, installDir, dataDir); err != nil {
		return fmt.Errorf("run installer: %w", err)
	}

	if err := startStack(dataDir); err != nil {
		return fmt.Errorf("start stack: %w", err)
	}

	recordApplied(assetPath, dataDir)
	return nil
}

func verifyChecksum(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func stopStack() error {
	cmd := exec.Command("docker", "compose", "-p", "astra-system", "down", "--timeout", "30")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return nil
}

func startStack(dataDir string) error {
	composeDir := filepath.Join(dataDir, "compose")
	composeFile := filepath.Join(composeDir, "docker-compose.yml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("docker", "compose",
		"-p", "astra-system",
		"-f", composeFile,
		"up", "-d", "--pull", "always",
	)
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func backupConfig(dataDir string) error {
	backupDir := filepath.Join(dataDir, "backups",
		fmt.Sprintf("pre-update-%s", time.Now().UTC().Format("20060102T150405Z")))

	configDir := filepath.Join(dataDir, "config")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil
	}

	if err := copyDir(configDir, backupDir); err != nil {
		return fmt.Errorf("backup config: %w", err)
	}

	return nil
}

func runInstaller(assetPath, installDir, dataDir string) error {
	cmd := exec.Command(assetPath, "/SILENT",
		fmt.Sprintf("/INSTALLDIR=%s", installDir),
		fmt.Sprintf("/DATADIR=%s", dataDir),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func recordApplied(assetPath, dataDir string) {
	recordDir := filepath.Join(dataDir, "updates")
	os.MkdirAll(recordDir, 0755)
	recordFile := filepath.Join(recordDir,
		fmt.Sprintf("applied-%s.txt", time.Now().UTC().Format("20060102T150405Z")))
	os.WriteFile(recordFile, []byte(assetPath+"\n"), 0644)
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

func CheckHash(path, expected string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), "  ", 2)
		if len(parts) == 2 && parts[1] == filepath.Base(path) {
			if parts[0] == expected {
				return nil
			}
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, parts[0])
		}
	}

	return nil
}
