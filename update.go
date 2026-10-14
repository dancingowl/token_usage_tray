package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// appVersion is supplied by release builds through -ldflags -X main.appVersion.
var appVersion = "dev"

type updateStatus struct {
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	Available      bool      `json:"available"`
	ReleaseURL     string    `json:"release_url"`
	Notes          string    `json:"notes"`
	CheckedAt      time.Time `json:"checked_at"`
	Error          string    `json:"error"`
}

const latestReleaseEndpoint = "https://api.github.com/repos/dancingowl/token_usage_tray/releases/latest"

type updateChecker struct {
	mu       sync.Mutex
	status   updateStatus
	inFlight chan struct{}
	client   *http.Client
	endpoint string
	retryAt  time.Time
}

var releaseChecker = &updateChecker{
	client:   &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	endpoint: latestReleaseEndpoint,
}

func currentUpdateStatus() updateStatus                { return releaseChecker.current(appVersion) }
func checkForUpdates(ctx context.Context) updateStatus { return releaseChecker.check(ctx, appVersion) }

func (c *updateChecker) current(version string) updateStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.status
	s.CurrentVersion = version
	s.Available = newerRelease(version, s.LatestVersion)
	return s
}

func (c *updateChecker) check(ctx context.Context, version string) updateStatus {
	c.mu.Lock()
	if pending := c.inFlight; pending != nil {
		c.mu.Unlock()
		select {
		case <-pending:
			return c.current(version)
		case <-ctx.Done():
			s := c.current(version)
			s.Error = "업데이트 확인이 취소되었거나 시간이 초과되었습니다."
			return s
		}
	}
	if time.Now().Before(c.retryAt) || !c.status.CheckedAt.IsZero() && time.Since(c.status.CheckedAt) < time.Minute {
		c.mu.Unlock()
		return c.current(version)
	}
	c.inFlight = make(chan struct{})
	c.mu.Unlock()
	result := c.fetch(ctx, version)
	c.mu.Lock()
	if result.Error != "" {
		// Preserve the last successful release details, while exposing this attempt's failure.
		result.LatestVersion = c.status.LatestVersion
		result.ReleaseURL = c.status.ReleaseURL
		result.Notes = c.status.Notes
		result.Available = newerRelease(version, result.LatestVersion)
	}
	c.status = result
	close(c.inFlight)
	c.inFlight = nil
	c.mu.Unlock()
	return result
}

func (c *updateChecker) fetch(ctx context.Context, version string) (s updateStatus) {
	s.CurrentVersion = version
	defer func() { s.CheckedAt = time.Now() }()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		s.Error = "업데이트 요청을 만들 수 없습니다."
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Usage-Tray-update-checker")
	resp, err := c.client.Do(req)
	if err != nil {
		s.Error = "업데이트 서버에 연결할 수 없습니다. 네트워크 연결을 확인해 주세요."
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		s.Error = "공개된 최신 릴리스를 찾을 수 없습니다. 아직 배포가 없거나 저장소에 접근할 수 없습니다."
		return
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden && (resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != "") {
		c.recordRateLimit(resp.Header)
		s.Error = "GitHub 요청 한도에 도달했습니다. 잠시 후 다시 확인해 주세요."
		return
	}
	if resp.StatusCode != http.StatusOK {
		s.Error = fmt.Sprintf("업데이트 서버가 요청을 처리하지 못했습니다 (HTTP %d).", resp.StatusCode)
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		s.Error = "업데이트 서버 응답을 읽을 수 없거나 응답이 너무 큽니다."
		return
	}
	var release struct {
		Tag        string `json:"tag_name"`
		URL        string `json:"html_url"`
		Body       string `json:"body"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if json.Unmarshal(data, &release) != nil || release.Draft || release.Prerelease || !validReleaseURL(release.URL) {
		s.Error = "업데이트 서버가 올바른 공개 릴리스 정보를 반환하지 않았습니다."
		return
	}
	if _, ok := parseReleaseVersion(release.Tag); !ok {
		s.Error = "최신 릴리스의 버전 형식을 확인할 수 없습니다."
		return
	}
	s.LatestVersion = release.Tag
	s.ReleaseURL = release.URL
	notes := []rune(release.Body)
	if len(notes) > 8000 {
		notes = append(notes[:8000], []rune("\n… (릴리스 페이지에서 전체 내용을 확인하세요.)")...)
	}
	s.Notes = string(notes)
	s.Available = newerRelease(version, release.Tag)
	return
}

// GitHub's rate-limit headers can require a wait longer than the normal minute.
func (c *updateChecker) recordRateLimit(header http.Header) {
	now := time.Now()
	retryAt := now.Add(time.Minute)
	if seconds, err := strconv.ParseInt(header.Get("Retry-After"), 10, 64); err == nil && seconds > 0 && seconds <= int64((365*24*time.Hour)/time.Second) {
		if until := now.Add(time.Duration(seconds) * time.Second); until.After(retryAt) {
			retryAt = until
		}
	} else if until, err := http.ParseTime(header.Get("Retry-After")); err == nil && until.After(retryAt) {
		retryAt = until
	}
	if header.Get("X-RateLimit-Remaining") == "0" {
		if epoch, err := strconv.ParseInt(header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			if until := time.Unix(epoch, 0); until.After(retryAt) {
				retryAt = until
			}
		}
	}
	c.mu.Lock()
	c.retryAt = retryAt
	c.mu.Unlock()
}

func validReleaseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	const prefix = "/dancingowl/token_usage_tray/releases/"
	if !strings.HasPrefix(u.Path, prefix) || len(u.Path) == len(prefix) {
		return false
	}
	for _, part := range strings.Split(u.Path, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return !strings.Contains(u.Path, "\\")
}

type releaseVersion struct {
	core [3]string
	pre  []string
}

var releaseVersionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

func parseReleaseVersion(value string) (v releaseVersion, ok bool) {
	m := releaseVersionPattern.FindStringSubmatch(value)
	if m == nil {
		return v, false
	}
	copy(v.core[:], m[1:4])
	if m[4] != "" {
		v.pre = strings.Split(m[4], ".")
		for _, p := range v.pre {
			if numericVersionPart(p) && len(p) > 1 && p[0] == '0' {
				return v, false
			}
		}
	}
	return v, true
}
func numericVersionPart(s string) bool {
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return s != ""
}
func compareVersionNumber(a, b string) int {
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return strings.Compare(a, b)
}
func newerRelease(current, latest string) bool {
	a, ok := parseReleaseVersion(current)
	if !ok {
		return false
	}
	b, ok := parseReleaseVersion(latest)
	if !ok {
		return false
	}
	for i := range a.core {
		if cmp := compareVersionNumber(a.core[i], b.core[i]); cmp != 0 {
			return cmp < 0
		}
	}
	if len(a.pre) == 0 {
		return false
	}
	if len(b.pre) == 0 {
		return true
	}
	for i := 0; i < len(a.pre) && i < len(b.pre); i++ {
		x, y := a.pre[i], b.pre[i]
		if x == y {
			continue
		}
		xn, yn := numericVersionPart(x), numericVersionPart(y)
		if xn && yn {
			return compareVersionNumber(x, y) < 0
		}
		if xn != yn {
			return xn
		}
		return x < y
	}
	return len(a.pre) < len(b.pre)
}
