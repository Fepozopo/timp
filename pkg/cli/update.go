package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

// detectLatestFallback queries the GitHub Releases API and returns a best-match
// release struct compatible with selfupdate.Release. It prefers published,
// non-prerelease releases with semver-compliant tag names and returns the highest
// semver it can find. If no suitable release is found it returns (nil, false, nil).
func detectLatestFallback(repo string) (*selfupdate.Release, bool, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, false, fmt.Errorf("github API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed reading github response: %w", err)
	}

	// Minimal struct to parse releases JSON
	var releases []struct {
		TagName    string `json:"tag_name"`
		Name       string `json:"name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, false, fmt.Errorf("failed to decode github releases: %w", err)
	}

	type candidate struct {
		ver      semver.Version
		tag      string
		assetURL string
		name     string
	}

	var candidates []candidate

	// regex to find semver substring like v1.2.3 or 1.2.3 inside tag name
	semverRe := regexp.MustCompile(`v?\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?`)

	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		tag := r.TagName
		match := semverRe.FindString(tag)
		if match == "" {
			// try the release name as a fallback
			match = semverRe.FindString(r.Name)
			if match == "" {
				continue
			}
		}
		// normalize to start with v if missing (semver.Parse accepts both but keep consistent)
		verStr := match
		// semver.Parse expects no leading 'v' for github.com/blang/semver, but it supports v-prefixed too.
		v, perr := semver.Parse(verStr)
		if perr != nil {
			// try stripping leading 'v'
			verStr = strings.TrimPrefix(match, "v")
			v, perr = semver.Parse(verStr)
			if perr != nil {
				continue
			}
		}
		assetURL := ""
		// pick first available asset (prefer ones that look like binaries)
		for _, a := range r.Assets {
			nameLower := strings.ToLower(a.Name)
			if strings.Contains(nameLower, "darwin") || strings.Contains(nameLower, "linux") || strings.Contains(nameLower, "windows") || strings.Contains(nameLower, "amd64") || strings.Contains(nameLower, "arm64") {
				assetURL = a.BrowserDownloadURL
				break
			}
			// fallback to first asset if nothing matches
			if assetURL == "" {
				assetURL = a.BrowserDownloadURL
			}
		}
		candidates = append(candidates, candidate{ver: v, tag: tag, assetURL: assetURL, name: r.Name})
	}

	if len(candidates) == 0 {
		return nil, false, nil
	}

	// pick the highest semver
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ver.GT(candidates[j].ver)
	})
	best := candidates[0]

	// Build a selfupdate.Release-like struct (only include fields present in the actual type)
	r := &selfupdate.Release{
		Version:  best.ver,
		AssetURL: best.assetURL,
	}
	return r, true, nil
}

func CheckForUpdates() error {
	const repo = "Fepozopo/timp"

	// Use the GitHub API fallback detector which is tolerant of tag naming.
	latest, found, err := detectLatestFallback(repo)
	fmt.Printf("Current version: %s\n", Version)
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}
	if latest == nil {
		fmt.Println("No release information available from GitHub.")
	}

	if latest != nil {
		fmt.Printf("Latest version: %s\n", latest.Version)
	}

	currentVer, parseErr := semver.Parse(Version)
	if parseErr != nil {
		// If the built Version isn't valid semver, continue but warn.
		fmt.Printf("warning: could not parse current version %q: %v\n", Version, parseErr)
	}

	// No release found or nil result -> nothing to do.
	if !found || latest == nil {
		fmt.Printf("No releases found for %s.\n", repo)
		return nil
	}

	// If same version -> up-to-date.
	if latest.Version.Equals(currentVer) {
		fmt.Printf("You are already running the latest version: %s.\n", currentVer)
		return nil
	}

	// If we don't have an asset URL, cannot update automatically.
	if latest.AssetURL == "" {
		fmt.Printf("A new version (%s) is available but there is no downloadable asset.\n", latest.Version)
		fmt.Println("Please visit the project releases page to download the new version.")
		return nil
	}

	// Prompt the user to confirm updating.
	answer, perr := PromptLine(fmt.Sprintf("A new version (%s) is available. Update now? (y/N): ", latest.Version))
	if perr != nil {
		return fmt.Errorf("failed reading input: %w", perr)
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Update cancelled.")
		return nil
	}

	fmt.Println("Updating...")
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate executable: %w", err)
	}

	if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	// Attempt to restart the process by replacing the current process image.
	argv := append([]string{exe}, os.Args[1:]...)
	if err := syscall.Exec(exe, argv, os.Environ()); err != nil {
		// Exec only returns on error. Try a fallback of starting the new binary as a child process.
		cmd := exec.Command(exe, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if startErr := cmd.Start(); startErr != nil {
			// If fallback also fails, report success but instruct user to restart manually.
			fmt.Printf("Updated to version %s, but failed to restart automatically: %v; fallback start error: %v\n", latest.Version, err, startErr)
			fmt.Println("Please restart the application manually.")
			return nil
		}
		// Successfully started the new process; exit the current one.
		os.Exit(0)
	}

	// If Exec succeeds, this process is replaced and the following lines won't run.
	return nil
}
