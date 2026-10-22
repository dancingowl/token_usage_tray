package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const automaticUpdateInterval = 6 * time.Hour
const automaticUpdateMaxBinary = 128 << 20
const automaticUpdateMaxChecksums = 64 << 10

type automaticUpdateStatus struct {
	Supported   bool      `json:"supported"`
	State       string    `json:"state"`
	CheckedAt   time.Time `json:"checked_at"`
	NextCheckAt time.Time `json:"next_check_at"`
	Version     string    `json:"version"`
	Error       string    `json:"error,omitempty"`
}

var automaticUpdateState struct {
	sync.Mutex
	status automaticUpdateStatus
}

func currentAutomaticUpdateStatus() automaticUpdateStatus {
	automaticUpdateState.Lock()
	defer automaticUpdateState.Unlock()
	s := automaticUpdateState.status
	s.Supported = automaticUpdateSupported() && stableUpdateVersion(appVersion)
	if s.State == "" {
		s.State = "disabled"
		if s.Supported {
			s.State = "idle"
		}
	}
	return s
}
func setAutomaticUpdateState(state, version string, err error) {
	automaticUpdateState.Lock()
	defer automaticUpdateState.Unlock()
	automaticUpdateState.status.State = state
	if version != "" {
		automaticUpdateState.status.Version = version
	}
	automaticUpdateState.status.Error = ""
	if err != nil {
		automaticUpdateState.status.Error = err.Error()
	}
}
func stableUpdateVersion(version string) bool {
	v, ok := parseReleaseVersion(version)
	return ok && len(v.pre) == 0 && !strings.Contains(version, "+") && strings.HasPrefix(version, "v")
}

type automaticReleaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}
type automaticRelease struct {
	Tag        string                  `json:"tag_name"`
	URL        string                  `json:"html_url"`
	Draft      bool                    `json:"draft"`
	Prerelease bool                    `json:"prerelease"`
	Assets     []automaticReleaseAsset `json:"assets"`
}

func exactUpdateURL(raw, expected string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "github.com" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.RawPath == "" && raw == expected
}
func (r automaticRelease) assets() (exe, sums automaticReleaseAsset, err error) {
	if !stableUpdateVersion(r.Tag) || r.Draft || r.Prerelease || !exactUpdateURL(r.URL, "https://github.com/dancingowl/token_usage_tray/releases/tag/"+r.Tag) {
		return exe, sums, errors.New("invalid official stable release")
	}
	for _, a := range r.Assets {
		if a.Name != "usage-tray.exe" && a.Name != "SHA256SUMS.txt" {
			continue
		}
		if !exactUpdateURL(a.URL, "https://github.com/dancingowl/token_usage_tray/releases/download/"+r.Tag+"/"+a.Name) {
			return exe, sums, errors.New("invalid release download URL")
		}
		limit := int64(automaticUpdateMaxBinary)
		if a.Name == "SHA256SUMS.txt" {
			limit = automaticUpdateMaxChecksums
		}
		if a.Size <= 0 || a.Size > limit {
			return exe, sums, errors.New("invalid release asset size")
		}
		if a.Digest != "" {
			if _, err := parseSHA256(strings.TrimPrefix(a.Digest, "sha256:")); err != nil || !strings.HasPrefix(a.Digest, "sha256:") {
				return exe, sums, errors.New("invalid asset digest")
			}
		}
		if a.Name == "usage-tray.exe" {
			if exe.Name != "" {
				return exe, sums, errors.New("duplicate executable asset")
			}
			exe = a
		} else {
			if sums.Name != "" {
				return exe, sums, errors.New("duplicate checksum asset")
			}
			sums = a
		}
	}
	if exe.Name == "" || sums.Name == "" {
		err = errors.New("release requires usage-tray.exe and SHA256SUMS.txt")
	}
	return
}
func parseSHA256(s string) (string, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != sha256.Size {
		return "", errors.New("invalid SHA256")
	}
	return strings.ToLower(s), nil
}
func checksumForExecutable(data []byte) (string, error) {
	var result string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != "usage-tray.exe" {
			continue
		}
		if result != "" {
			return "", errors.New("duplicate executable checksum")
		}
		h, err := parseSHA256(fields[0])
		if err != nil {
			return "", err
		}
		result = h
	}
	if result == "" {
		return "", errors.New("executable checksum missing")
	}
	return result, nil
}
func updateDownloadClient() *http.Client {
	return &http.Client{Timeout: 3 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 4 || req.URL.Scheme != "https" || req.URL.User != nil || (req.URL.Host != "release-assets.githubusercontent.com" && req.URL.Host != "objects.githubusercontent.com") {
			return errors.New("unsafe release asset redirect")
		}
		return nil
	}}
}
func readUpdateResponse(ctx context.Context, client *http.Client, raw string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Usage-Tray-auto-update")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden && (resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != "") {
			releaseChecker.recordRateLimit(resp.Header)
		}
		return nil, fmt.Errorf("release server HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > max {
		return nil, errors.New("release response too large")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errors.New("release response too large")
	}
	return b, nil
}
func fetchAutomaticRelease(ctx context.Context) (automaticRelease, error) {
	var r automaticRelease
	b, err := readUpdateResponse(ctx, releaseChecker.client, latestReleaseEndpoint, 1<<20)
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal(b, &r); err != nil {
		return r, errors.New("invalid release response")
	}
	return r, nil
}
func verifiedUpdateBinary(ctx context.Context, r automaticRelease) ([]byte, string, error) {
	exe, sums, err := r.assets()
	if err != nil {
		return nil, "", err
	}
	client := updateDownloadClient()
	checksums, err := readUpdateResponse(ctx, client, sums.URL, automaticUpdateMaxChecksums)
	if err != nil {
		return nil, "", err
	}
	if int64(len(checksums)) != sums.Size {
		return nil, "", errors.New("checksum asset size mismatch")
	}
	if err = verifyAssetDigest(checksums, sums.Digest); err != nil {
		return nil, "", err
	}
	hash, err := checksumForExecutable(checksums)
	if err != nil {
		return nil, "", err
	}
	b, err := readUpdateResponse(ctx, client, exe.URL, automaticUpdateMaxBinary)
	if err != nil {
		return nil, "", err
	}
	if int64(len(b)) != exe.Size {
		return nil, "", errors.New("executable size mismatch")
	}
	if err = verifyAssetDigest(b, "sha256:"+hash); err != nil {
		return nil, "", err
	}
	if err = verifyAssetDigest(b, exe.Digest); err != nil {
		return nil, "", err
	}
	return b, hash, nil
}
func verifyAssetDigest(b []byte, digest string) error {
	if digest == "" {
		return nil
	}
	h := sha256.Sum256(b)
	if digest != "sha256:"+hex.EncodeToString(h[:]) {
		return errors.New("release SHA256 mismatch")
	}
	return nil
}
func updateWorkDir(exe string) string {
	// Windows may report the same file using either an 8.3 alias or its long
	// name. Every process must hash the physical executable's canonical path.
	if canonical, err := canonicalUpdatePath(exe); err == nil {
		exe = canonical
	}
	h := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(exe))))
	return filepath.Join(filepath.Dir(exe), ".usage-tray-update-"+hex.EncodeToString(h[:8]))
}
func canonicalUpdatePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}
func sameUpdatePath(a, b string) bool {
	a, err := canonicalUpdatePath(a)
	if err != nil {
		return false
	}
	b, err = canonicalUpdatePath(b)
	return err == nil && strings.EqualFold(a, b)
}
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, automaticUpdateMaxBinary+1))
	if err != nil {
		return "", err
	}
	if n > automaticUpdateMaxBinary {
		return "", errors.New("executable too large")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
