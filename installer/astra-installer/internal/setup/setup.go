package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const AppName = "Astra-System"

func Install(installDir, dataDir, channel string, silent bool) error {
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	if !silent {
		fmt.Println("  ✓ Directories created")
	}

	updaterSrc := filepath.Join(installDir, "bin", "astra-updater.exe")
	if _, err := os.Stat(updaterSrc); err == nil {
		cmd := exec.Command(updaterSrc, "install",
			"--data-dir", dataDir,
			"--install-dir", installDir,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("register update service: %w", err)
		}
		if !silent {
			fmt.Println("  ✓ Update agent service registered")
		}
	}

	return nil
}
