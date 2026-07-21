//go:build !windows

package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const AppName = "Astra-System"

func Install(installDir, dataDir, channel string, silent bool) error {
	binPath := filepath.Join(installDir, "bin", "astra-installer")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		binPath = "/usr/local/bin/astra-installer"
	}

	switch runtime.GOOS {
	case "darwin":
		return installLaunchDaemon(binPath, dataDir, channel, silent)
	case "linux":
		return installSystemd(binPath, dataDir, channel, silent)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// ── Privilege helpers ──────────────────────────────────────────

func isPrivileged() bool {
	return os.Geteuid() == 0
}

func writePrivileged(path string, data []byte, perm os.FileMode) error {
	if isPrivileged() {
		return os.WriteFile(path, data, perm)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	cmd := exec.Command("sudo", "mv", tmpPath, path)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("sudo mv: %s: %w", string(out), err)
	}
	cmd = exec.Command("sudo", "chmod", fmt.Sprintf("%o", perm), path)
	cmd.Run()
	return nil
}

func execPrivileged(name string, args ...string) error {
	if isPrivileged() {
		return run(name, args...)
	}
	// Try non-interactive sudo first
	if err := run("sudo", append([]string{"-n", name}, args...)...); err == nil {
		return nil
	}
	// Fall back to interactive sudo
	return run("sudo", append([]string{name}, args...)...)
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ── macOS LaunchDaemon ─────────────────────────────────────────

func installLaunchDaemon(binPath, dataDir, channel string, silent bool) error {
	plistPath := "/Library/LaunchDaemons/com.astra-system.updater.plist"

	if _, err := os.Stat(filepath.Dir(plistPath)); os.IsNotExist(err) {
		if !silent {
			fmt.Println("  ! /Library/LaunchDaemons does not exist — skipping auto-update")
		}
		return nil
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.astra-system.updater</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>--silent</string>
        <string>--data-dir</string>
        <string>%s</string>
        <string>--channel</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StartInterval</key>
    <integer>21600</integer>
    <key>StandardOutPath</key>
    <string>%s/logs/updater.log</string>
    <key>StandardErrorPath</key>
    <string>%s/logs/updater.log</string>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`, binPath, dataDir, channel, dataDir, dataDir)

	if !isPrivileged() && !silent {
		fmt.Println("  ! Administrator privileges required for LaunchDaemon")
		fmt.Println("  ! You may be prompted for your password (sudo)")
	}

	if err := writePrivileged(plistPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	_ = execPrivileged("launchctl", "unload", plistPath)
	if err := execPrivileged("launchctl", "load", plistPath); err != nil {
		return fmt.Errorf("load launchd plist: %w", err)
	}

	if !silent {
		fmt.Println("  ✓ LaunchDaemon installed — checks for updates every 6 hours")
	}
	return nil
}

// ── Linux systemd ──────────────────────────────────────────────

func installSystemd(binPath, dataDir, channel string, silent bool) error {
	unitDir := "/etc/systemd/system"
	servicePath := filepath.Join(unitDir, "astra-updater.service")
	timerPath := filepath.Join(unitDir, "astra-updater.timer")

	if _, err := os.Stat(unitDir); os.IsNotExist(err) {
		if !silent {
			fmt.Println("  ! /etc/systemd/system does not exist — skipping auto-update")
		}
		return nil
	}

	serviceContent := fmt.Sprintf(`[Unit]
Description=Astra-System Update Agent
After=network-online.target docker.service
Wants=network-online.target docker.service

[Service]
Type=oneshot
ExecStart=%s --silent --data-dir %s --channel %s
`, binPath, dataDir, channel)

	timerContent := `[Unit]
Description=Astra-System Update Timer (every 6 hours)

[Timer]
OnBootSec=5min
OnUnitActiveSec=6h
Persistent=true

[Install]
WantedBy=timers.target
`

	if !isPrivileged() && !silent {
		fmt.Println("  ! Administrator privileges required for systemd units")
		fmt.Println("  ! You may be prompted for your password (sudo)")
	}

	if err := writePrivileged(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("write service: %w", err)
	}
	if err := writePrivileged(timerPath, []byte(timerContent), 0644); err != nil {
		return fmt.Errorf("write timer: %w", err)
	}

	_ = execPrivileged("systemctl", "daemon-reload")
	_ = execPrivileged("systemctl", "enable", "astra-updater.timer")
	_ = execPrivileged("systemctl", "start", "astra-updater.timer")

	if !silent {
		fmt.Println("  ✓ Systemd timer installed — checks for updates every 6 hours")
	}
	return nil
}

// ── Remove (uninstall) ─────────────────────────────────────────

func Remove(dataDir string) error {
	if !isPrivileged() {
		fmt.Println("  ! Skipping system service cleanup (not running as root)")
		fmt.Println("  ! To remove the update service, run:")
		fmt.Println("  !   sudo astra-installer --remove")
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		plistPath := "/Library/LaunchDaemons/com.astra-system.updater.plist"
		run("launchctl", "unload", plistPath)
		os.Remove(plistPath)
	case "linux":
		run("systemctl", "stop", "astra-updater.timer")
		run("systemctl", "disable", "astra-updater.timer")
		os.Remove("/etc/systemd/system/astra-updater.service")
		os.Remove("/etc/systemd/system/astra-updater.timer")
		run("systemctl", "daemon-reload")
	}
	return nil
}
