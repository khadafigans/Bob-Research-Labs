package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const Repo = "Autumn-27/artex"

// latestURL  GitHub ""。 prerelease  draft。
const latestURL = "https://api.github.com/repos/" + Repo + "/releases/latest"

// allowedHosts 。 checkRedirect，
var allowedHosts = map[string]bool{
	"api.github.com":                       true,
	"github.com":                           true,
	"objects.githubusercontent.com":        true, // release 
	"release-assets.githubusercontent.com": true,
	"raw.githubusercontent.com":            true,
}

// Release  GitHub Release 。
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []Asset   `json:"assets"`
}

// Asset  Release 。
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// NewClient  GitHub  HTTP 。proxy 。
//
//  Transport： TLS ，
//  InsecureSkipVerify 。
func NewClient(proxy string) *http.Client {
	tr := &http.Transport{
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 15 * time.Second,
	}
	if p := strings.TrimSpace(proxy); p != "" {
		if pu, err := url.Parse(p); err == nil {
			tr.Proxy = http.ProxyURL(pu)
		}
	}
	return &http.Client{
		Transport: tr,
		Timeout:   30 * time.Minute, // ，
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("   ")
			}
			return checkURL(req.URL)
		},
	}
}

// checkURL  https + 。
func checkURL(u *url.URL) error {
	if u.Scheme != "https" {
		return fmt.Errorf("HTTPS    : %s", u.Scheme+"://"+u.Host)
	}
	if !allowedHosts[strings.ToLower(u.Hostname())] {
		return fmt.Errorf("GitHub    : %s", u.Hostname())
	}
	return nil
}

func FetchLatest(ctx context.Context, c *http.Client) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return nil, err
	}
	if err := checkURL(req.URL); err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "artex-selfupdate")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub   (      ): %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusTooManyRequests:
		//  GitHub API  IP  60 ， IP 。
		return nil, fmt.Errorf("GitHub API   ( 60).    ")
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf(" %s     ", Repo)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("GitHub   %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("Release  : %w", err)
	}
	if strings.TrimSpace(rel.TagName) == "" {
		return nil, fmt.Errorf("Release  tag  ")
	}
	return &rel, nil
}

// AssetName ， build.sh  package_binary ：
// artex-<>-<os>-<arch>.zip（ v ）。
func AssetName(tag, goos, goarch string) string {
	return fmt.Sprintf("artex-%s-%s-%s.zip", strings.TrimPrefix(tag, "v"), goos, goarch)
}

// FindAsset  Release 。
func (r *Release) FindAsset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if strings.EqualFold(a.Name, name) {
			return a, true
		}
	}
	return Asset{}, false
}
