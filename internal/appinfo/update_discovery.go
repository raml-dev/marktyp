package appinfo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRepositoryURL = "https://github.com/raml-dev/marktyp"
	releasesPerPage      = 20
	maxResponseSize      = 2 << 20
)

// DiscoveryClient finds newer Marktyp releases on GitHub.
type DiscoveryClient struct {
	client   *http.Client
	endpoint string
}

// GitHubResponse is the update payload exposed to the frontend.
type GitHubResponse struct {
	Release *GitHubRelease `json:"Release"`
}

// GitHubRelease contains the release details used by the update UI.
type GitHubRelease struct {
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
	HTMLURL    string `json:"html_url"`
	UpdatedAt  string `json:"updated_at"`
	Name       string `json:"name"`
	TagName    string `json:"tag_name"`
	PreRelease bool   `json:"prerelease"`
}

// NewDiscoveryClient creates a release client for a GitHub repository URL.
func NewDiscoveryClient(repositoryURL string) *DiscoveryClient {
	owner, repo := parseRepository(repositoryURL)
	return &DiscoveryClient{
		client:   &http.Client{Timeout: 30 * time.Second},
		endpoint: fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", owner, repo, releasesPerPage),
	}
}

// GetUpdatesFromRepo returns the newest eligible release newer than currentVersion.
func (client *DiscoveryClient) GetUpdatesFromRepo(
	ctx context.Context,
	currentVersion string,
	includePrereleases bool,
) (*GitHubResponse, error) {
	if client == nil || client.client == nil {
		return nil, errors.New("discovery client not initialized")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "marktyp-update-checker")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	response, err := client.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("request failed: %s - %s", response.Status, strings.TrimSpace(string(body)))
	}

	var releases []GitHubRelease
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseSize))
	if err := decoder.Decode(&releases); err != nil {
		return nil, err
	}

	latest := findLatestRelease(releases, currentVersion, includePrereleases)
	if latest == nil {
		return nil, nil
	}

	return &GitHubResponse{Release: latest}, nil
}

type parsedVersion struct {
	major            int
	minor            int
	patch            int
	prerelease       string
	prereleasePrefix string
	prereleaseNumber int
	hasPreNumber     bool
}

var prereleasePattern = regexp.MustCompile(`^([A-Za-z]+)[\.-]?(\d+)?$`)

func parseRepository(repositoryURL string) (owner string, repo string) {
	raw := strings.TrimSpace(repositoryURL)
	if raw == "" {
		raw = defaultRepositoryURL
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "raml-dev", "marktyp"
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return "raml-dev", "marktyp"
	}

	return parts[0], strings.TrimSuffix(parts[1], ".git")
}

func shouldIncludePrereleases(currentVersion string, configured bool) bool {
	if configured {
		return true
	}

	current := strings.TrimSpace(strings.TrimPrefix(currentVersion, "v"))
	if current == "" || strings.EqualFold(current, "dev") {
		return true
	}

	parsed, ok := parseVersion(current)
	return ok && parsed.prerelease != ""
}

func findLatestRelease(releases []GitHubRelease, currentVersion string, includePrereleases bool) *GitHubRelease {
	includePrereleases = shouldIncludePrereleases(currentVersion, includePrereleases)

	var latest *GitHubRelease
	for index := range releases {
		release := releases[index]
		if strings.TrimSpace(release.TagName) == "" || release.PreRelease && !includePrereleases {
			continue
		}
		if compareVersionStrings(currentVersion, release.TagName) >= 0 {
			continue
		}
		if latest == nil || compareVersionStrings(release.TagName, latest.TagName) > 0 {
			selected := release
			latest = &selected
		}
	}

	return latest
}

func compareVersionStrings(left, right string) int {
	leftValue := strings.TrimSpace(left)
	rightValue := strings.TrimSpace(right)
	if strings.EqualFold(leftValue, "dev") || leftValue == "" {
		if rightValue == "" {
			return 0
		}
		return -1
	}

	leftVersion, leftOK := parseVersion(leftValue)
	rightVersion, rightOK := parseVersion(rightValue)
	switch {
	case leftOK && rightOK:
		return compareParsedVersion(leftVersion, rightVersion)
	case leftOK:
		return 1
	case rightOK:
		return -1
	default:
		return strings.Compare(leftValue, rightValue)
	}
}

func parseVersion(raw string) (parsedVersion, bool) {
	value := strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	if value == "" {
		return parsedVersion{}, false
	}

	base := value
	prerelease := ""
	if index := strings.Index(base, "+"); index >= 0 {
		base = base[:index]
	}
	if index := strings.Index(base, "-"); index >= 0 {
		prerelease = base[index+1:]
		base = base[:index]
	}

	parts := strings.Split(base, ".")
	if len(parts) != 3 {
		return parsedVersion{}, false
	}

	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	patch, patchErr := strconv.Atoi(parts[2])
	if majorErr != nil || minorErr != nil || patchErr != nil {
		return parsedVersion{}, false
	}

	parsed := parsedVersion{major: major, minor: minor, patch: patch, prerelease: prerelease}
	if matches := prereleasePattern.FindStringSubmatch(prerelease); len(matches) == 3 {
		parsed.prereleasePrefix = strings.ToLower(matches[1])
		if matches[2] != "" {
			if number, err := strconv.Atoi(matches[2]); err == nil {
				parsed.prereleaseNumber = number
				parsed.hasPreNumber = true
			}
		}
	}

	return parsed, true
}

func compareParsedVersion(left, right parsedVersion) int {
	for _, pair := range [][2]int{{left.major, right.major}, {left.minor, right.minor}, {left.patch, right.patch}} {
		if pair[0] > pair[1] {
			return 1
		}
		if pair[0] < pair[1] {
			return -1
		}
	}

	if left.prerelease == "" && right.prerelease != "" {
		return 1
	}
	if left.prerelease != "" && right.prerelease == "" {
		return -1
	}
	if left.prerelease == "" {
		return 0
	}
	if left.prereleasePrefix != right.prereleasePrefix {
		return strings.Compare(left.prereleasePrefix, right.prereleasePrefix)
	}
	if left.hasPreNumber && right.hasPreNumber {
		if left.prereleaseNumber > right.prereleaseNumber {
			return 1
		}
		if left.prereleaseNumber < right.prereleaseNumber {
			return -1
		}
		return 0
	}

	return strings.Compare(left.prerelease, right.prerelease)
}
