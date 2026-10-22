package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testAutomaticRelease() automaticRelease {
	return automaticRelease{Tag: "v2.0.0", URL: "https://github.com/dancingowl/token_usage_tray/releases/tag/v2.0.0", Assets: []automaticReleaseAsset{
		{Name: "usage-tray.exe", URL: "https://github.com/dancingowl/token_usage_tray/releases/download/v2.0.0/usage-tray.exe", Size: 32},
		{Name: "SHA256SUMS.txt", URL: "https://github.com/dancingowl/token_usage_tray/releases/download/v2.0.0/SHA256SUMS.txt", Size: 80},
	}}
}
func TestAutomaticReleaseTrustBoundary(t *testing.T) {
	if _, _, err := testAutomaticRelease().assets(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*automaticRelease){
		func(r *automaticRelease) { r.Draft = true }, func(r *automaticRelease) { r.Prerelease = true },
		func(r *automaticRelease) { r.Tag = "v2.0.0-rc.1" }, func(r *automaticRelease) { r.Tag = "v2.0.0+dev" },
		func(r *automaticRelease) { r.URL = strings.Replace(r.URL, "v2.0.0", "v1.0.0", 1) },
		func(r *automaticRelease) {
			r.Assets[0].URL = strings.Replace(r.Assets[0].URL, "github.com", "github.com.evil.test", 1)
		},
		func(r *automaticRelease) { r.Assets[0].URL += "?download=1" },
		func(r *automaticRelease) {
			r.Assets[0].URL = strings.Replace(r.Assets[0].URL, "v2.0.0", "%76%32.0.0", 1)
		},
		func(r *automaticRelease) { r.Assets[0].Size = automaticUpdateMaxBinary + 1 },
		func(r *automaticRelease) { r.Assets[1].Size = 0 },
		func(r *automaticRelease) { r.Assets[0].Digest = "md5:" + strings.Repeat("a", 64) },
		func(r *automaticRelease) { r.Assets = append(r.Assets, r.Assets[0]) },
		func(r *automaticRelease) { r.Assets = r.Assets[:1] },
	} {
		r := testAutomaticRelease()
		mutate(&r)
		if _, _, err := r.assets(); err == nil {
			t.Fatalf("accepted hostile release: %+v", r)
		}
	}
}
func TestAutomaticChecksumVerification(t *testing.T) {
	b := []byte("official binary")
	sum := sha256.Sum256(b)
	hash := hex.EncodeToString(sum[:])
	for _, line := range []string{hash + "  usage-tray.exe\n", hash + " *usage-tray.exe\r\n"} {
		got, err := checksumForExecutable([]byte(line))
		if err != nil || got != hash {
			t.Fatalf("%q %v", got, err)
		}
	}
	for _, line := range []string{hash + "  ../usage-tray.exe", hash + " usage-tray.exe\n" + hash + " usage-tray.exe", strings.Repeat("g", 64) + " usage-tray.exe"} {
		if _, err := checksumForExecutable([]byte(line)); err == nil {
			t.Fatalf("accepted bad checksum %q", line)
		}
	}
	if err := verifyAssetDigest(b, "sha256:"+hash); err != nil {
		t.Fatal(err)
	}
	if err := verifyAssetDigest(append(b, 'x'), "sha256:"+hash); err == nil {
		t.Fatal("accepted modified executable")
	}
}
func TestAutomaticDownloadBounds(t *testing.T) {
	for _, tt := range []struct {
		body   string
		header bool
	}{{"abcd", true}, {"abcd", false}, {"ab", false}} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !tt.header {
				w.(http.Flusher).Flush()
			}
			w.Write([]byte(tt.body))
		}))
		b, err := readUpdateResponse(context.Background(), srv.Client(), srv.URL, 3)
		srv.Close()
		if len(tt.body) > 3 && err == nil {
			t.Fatal("oversized response accepted")
		}
		if len(tt.body) <= 3 && (err != nil || string(b) != tt.body) {
			t.Fatalf("valid download: %v", err)
		}
	}
	client := updateDownloadClient()
	for _, raw := range []string{"http://release-assets.githubusercontent.com/file", "https://release-assets.githubusercontent.com.evil.test/file", "https://github.com/file", "https://user@release-assets.githubusercontent.com/file"} {
		req, _ := http.NewRequest("GET", raw, nil)
		if client.CheckRedirect(req, nil) == nil {
			t.Fatalf("unsafe redirect %s", raw)
		}
	}
}
func TestAutomaticDevelopmentAndPrereleaseSkip(t *testing.T) {
	for _, v := range []string{"dev", "0.2.0", "v0.2.0-rc.1", "v0.2.0+dirty", "unknown"} {
		if stableUpdateVersion(v) {
			t.Fatalf("automatic update enabled for %s", v)
		}
	}
	if !stableUpdateVersion("v0.2.0") {
		t.Fatal("stable disabled")
	}
}
