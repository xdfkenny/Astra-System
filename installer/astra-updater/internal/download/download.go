package download

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/astra-service/astra-updater/internal/check"
)

var HTTPClient = &http.Client{Timeout: 30 * time.Minute}

type Result struct {
	Path     string
	Checksum string
}

func Asset(release *check.ReleaseInfo, dataDir string) (string, string, error) {
	if len(release.Assets) == 0 {
		return "", "", fmt.Errorf("no assets in release %s", release.TagName)
	}

	asset := release.Assets[0]
	stagingDir := filepath.Join(dataDir, "staging")
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return "", "", fmt.Errorf("create staging dir: %w", err)
	}

	destPath := filepath.Join(stagingDir, asset.Name)
	if _, err := os.Stat(destPath); err == nil {
		checksum, err := fileChecksum(destPath)
		if err == nil {
			return destPath, checksum, nil
		}
	}

	tmpPath := destPath + ".tmp"
	if err := downloadFile(asset.BrowserDownloadURL, tmpPath); err != nil {
		return "", "", fmt.Errorf("download %s: %w", asset.Name, err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", "", fmt.Errorf("finalize download: %w", err)
	}

	checksum, err := fileChecksum(destPath)
	if err != nil {
		return "", "", fmt.Errorf("checksum: %w", err)
	}

	return destPath, checksum, nil
}

func downloadFile(url, dest string) error {
	resp, err := HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(dest)
		return fmt.Errorf("write file: %w", err)
	}

	if resp.ContentLength > 0 && written != resp.ContentLength {
		os.Remove(dest)
		return fmt.Errorf("incomplete download: %d/%d bytes", written, resp.ContentLength)
	}

	return nil
}

func fileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
