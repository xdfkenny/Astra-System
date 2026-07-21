package check

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const githubAPI = "https://api.github.com/repos/%s/%s/releases"

var (
	RepoOwner = "astra-service"
	RepoName  = "Astra-System"
	HTTPClient = &http.Client{Timeout: 30 * time.Second}
)

type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	PublishedAt string        `json:"published_at"`
	Body        string        `json:"body"`
	Prerelease  bool          `json:"prerelease"`
	Assets      []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ReleaseInfo struct {
	TagName     string
	Version     string
	Channel     string
	PublishedAt time.Time
	Assets      []GitHubAsset
}

func LatestRelease(channel, currentVersion string) (*ReleaseInfo, error) {
	url := fmt.Sprintf(githubAPI, RepoOwner, RepoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	if perPage := os.Getenv("GITHUB_TOKEN"); perPage != "" {
		req.Header.Set("Authorization", "Bearer "+perPage)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API: %s", resp.Status)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode releases: %w", err)
	}

	for _, r := range releases {
		info := parseRelease(r)
		if info.Channel != channel {
			continue
		}

		if compareVersions(info.Version, currentVersion) <= 0 {
			continue
		}

		installerAsset := findInstallerAsset(r.Assets)
		if installerAsset == nil {
			continue
		}

		return &ReleaseInfo{
			TagName:     r.TagName,
			Version:     info.Version,
			Channel:     info.Channel,
			PublishedAt: info.PublishedAt,
			Assets:      []GitHubAsset{*installerAsset},
		}, nil
	}

	return nil, nil
}

func parseRelease(r GitHubRelease) ReleaseInfo {
	info := ReleaseInfo{
		TagName: r.TagName,
		Channel: "stable",
	}
	info.PublishedAt, _ = time.Parse(time.RFC3339, r.PublishedAt)

	tag := strings.TrimPrefix(r.TagName, "v")
	if idx := strings.LastIndex(tag, "-"); idx > 0 {
		suffix := tag[idx+1:]
		switch suffix {
		case "stable", "beta", "canary":
			info.Channel = suffix
			info.Version = tag[:idx]
		default:
			info.Version = tag
		}
	} else {
		info.Version = tag
	}

	return info
}

func findInstallerAsset(assets []GitHubAsset) *GitHubAsset {
	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if strings.Contains(name, "astra") && strings.HasSuffix(name, ".exe") {
			return &a
		}
	}
	return nil
}

func compareVersions(a, b string) int {
	va := parseSemver(a)
	vb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if va[i] != vb[i] {
			if va[i] > vb[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	parts := strings.Split(v, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		fmt.Sscanf(parts[i], "%d", &result[i])
	}
	return result
}
