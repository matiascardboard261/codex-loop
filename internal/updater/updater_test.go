package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compozy/codex-loop/internal/loop"
)

func TestUpgradeDownloadsVerifiesInstallsAndRefreshesMarketplace(t *testing.T) {
	archiveName, binaryName, err := releaseArchiveName("0.1.1", "darwin", "arm64")
	if err != nil {
		t.Fatalf("archive name: %v", err)
	}
	archiveContent := tarGzipArchive(t, binaryName, []byte("new-binary\n"))
	checksum := sha256.Sum256(archiveContent)
	checksumsContent := []byte(fmt.Sprintf("%s  %s\n", hex.EncodeToString(checksum[:]), archiveName))

	server := releaseServer(t, "v0.1.1", map[string][]byte{
		archiveName:     archiveContent,
		"checksums.txt": checksumsContent,
	})

	codexHome := filepath.Join(t.TempDir(), ".codex-home")
	paths, err := loop.NewPaths(codexHome)
	if err != nil {
		t.Fatalf("new paths: %v", err)
	}
	targetBinary := filepath.Join(t.TempDir(), "bin", "codex-loop")
	if err := os.MkdirAll(filepath.Dir(targetBinary), 0o755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	if err := os.WriteFile(targetBinary, []byte("old-binary\n"), 0o755); err != nil {
		t.Fatalf("write old target: %v", err)
	}
	codexLog := filepath.Join(t.TempDir(), "codex-args.log")
	fakeCodex := writeFakeCodex(t, codexLog)

	messages, err := Upgrade(context.Background(), paths, Options{
		Version:      "v0.1.1",
		APIBaseURL:   server.URL,
		TargetBinary: targetBinary,
		CodexBinary:  fakeCodex,
		GOOS:         "darwin",
		GOARCH:       "arm64",
	})
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	if got := readFile(t, targetBinary); got != "new-binary\n" {
		t.Fatalf("expected target binary updated, got %q", got)
	}
	if got := readFile(t, paths.RuntimeBinaryPath()); got != "new-binary\n" {
		t.Fatalf("expected managed runtime updated, got %q", got)
	}
	if got := readFile(t, codexLog); !strings.Contains(got, "plugin marketplace add compozy/codex-loop --ref v0.1.1") {
		t.Fatalf("expected marketplace add command, got %q", got)
	}
	if got := readFile(t, codexLog); !strings.Contains(got, "plugin marketplace upgrade codex-loop-plugins") {
		t.Fatalf("expected marketplace upgrade command, got %q", got)
	}

	joined := strings.Join(messages, "\n")
	assertContains(t, joined, "Downloaded codex-loop v0.1.1")
	assertContains(t, joined, "Verified checksum")
	assertContains(t, joined, "Updated CLI binary")
	assertContains(t, joined, "Installed runtime binary")
	assertContains(t, joined, "Refreshed Codex plugin marketplace")
}

func TestUpgradeRejectsChecksumMismatchBeforeReplacingBinary(t *testing.T) {
	t.Parallel()

	archiveName, binaryName, err := releaseArchiveName("0.1.1", "darwin", "arm64")
	if err != nil {
		t.Fatalf("archive name: %v", err)
	}
	archiveContent := tarGzipArchive(t, binaryName, []byte("new-binary\n"))
	checksumsContent := []byte(strings.Repeat("0", sha256.Size*2) + "  " + archiveName + "\n")
	server := releaseServer(t, "v0.1.1", map[string][]byte{
		archiveName:     archiveContent,
		"checksums.txt": checksumsContent,
	})

	codexHome := filepath.Join(t.TempDir(), ".codex-home")
	paths, err := loop.NewPaths(codexHome)
	if err != nil {
		t.Fatalf("new paths: %v", err)
	}
	targetBinary := filepath.Join(t.TempDir(), "bin", "codex-loop")
	if err := os.MkdirAll(filepath.Dir(targetBinary), 0o755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	if err := os.WriteFile(targetBinary, []byte("old-binary\n"), 0o755); err != nil {
		t.Fatalf("write old target: %v", err)
	}

	_, err = Upgrade(context.Background(), paths, Options{
		Version:         "0.1.1",
		APIBaseURL:      server.URL,
		TargetBinary:    targetBinary,
		GOOS:            "darwin",
		GOARCH:          "arm64",
		SkipMarketplace: true,
	})
	if err == nil {
		t.Fatal("expected checksum error")
	}
	assertContains(t, err.Error(), "checksum mismatch")
	if got := readFile(t, targetBinary); got != "old-binary\n" {
		t.Fatalf("expected target binary preserved, got %q", got)
	}
	if _, err := os.Stat(paths.RuntimeBinaryPath()); !os.IsNotExist(err) {
		t.Fatalf("expected runtime not installed, stat err: %v", err)
	}
}

func TestReleaseArchiveName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		goos       string
		goarch     string
		archive    string
		binaryName string
	}{
		{
			name:       "darwin arm64",
			goos:       "darwin",
			goarch:     "arm64",
			archive:    "codex-loop_0.1.1_darwin_arm64.tar.gz",
			binaryName: "codex-loop",
		},
		{
			name:       "linux amd64",
			goos:       "linux",
			goarch:     "amd64",
			archive:    "codex-loop_0.1.1_linux_x86_64.tar.gz",
			binaryName: "codex-loop",
		},
		{
			name:       "windows amd64",
			goos:       "windows",
			goarch:     "amd64",
			archive:    "codex-loop_0.1.1_windows_x86_64.zip",
			binaryName: "codex-loop.exe",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			archive, binaryName, err := releaseArchiveName("0.1.1", tt.goos, tt.goarch)
			if err != nil {
				t.Fatalf("release archive name: %v", err)
			}
			if archive != tt.archive {
				t.Fatalf("expected archive %q, got %q", tt.archive, archive)
			}
			if binaryName != tt.binaryName {
				t.Fatalf("expected binary %q, got %q", tt.binaryName, binaryName)
			}
		})
	}
}

func releaseServer(t *testing.T, tag string, assets map[string][]byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/repos/compozy/codex-loop/releases/") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"tag_name":%q,"assets":[`, tag)
			index := 0
			for name := range assets {
				if index > 0 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, `{"name":%q,"browser_download_url":"%s/assets/%s"}`, name, serverURL(r), name)
				index++
			}
			fmt.Fprint(w, `]}`)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			name := strings.TrimPrefix(r.URL.Path, "/assets/")
			content, ok := assets[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
			if _, err := w.Write(content); err != nil {
				t.Errorf("write response: %v", err)
			}
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func tarGzipArchive(t *testing.T, binaryName string, binaryContent []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{
		Name: "codex-loop/" + binaryName,
		Mode: 0o755,
		Size: int64(len(binaryContent)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := io.Copy(tarWriter, bytes.NewReader(binaryContent)); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buffer.Bytes()
}

func writeFakeCodex(t *testing.T, logPath string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codex")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$CODEX_LOOP_TEST_CODEX_LOG\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv("CODEX_LOOP_TEST_CODEX_LOG", logPath)
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}
