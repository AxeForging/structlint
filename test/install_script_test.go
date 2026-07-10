package test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallScript_InstallsVerifiedPinnedRelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX installer")
	}
	osName, archName := runtime.GOOS, runtime.GOARCH
	if (osName != "linux" && osName != "darwin") || (archName != "amd64" && archName != "arm64" && archName != "386" && archName != "arm") {
		t.Skip("unsupported fixture platform")
	}
	tmp := t.TempDir()
	payload := filepath.Join(tmp, "payload")
	if err := os.Mkdir(payload, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(payload, "structlint"), []byte("#!/bin/sh\nprintf 'installed structlint\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	asset := "structlint-" + osName + "-" + archName + ".tar.gz"
	archive := filepath.Join(tmp, asset)
	if output, err := exec.Command("tar", "-czf", archive, "-C", payload, "structlint").CombinedOutput(); err != nil {
		t.Fatalf("archive: %v\n%s", err, output)
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	checksums := filepath.Join(tmp, "checksums.txt")
	if err := os.WriteFile(checksums, []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(data), asset)), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeBin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeCurl := `#!/bin/sh
set -eu
while [ "$#" -gt 0 ]; do case "$1" in -o) out="$2"; shift 2 ;; -*) shift ;; *) url="$1"; shift ;; esac; done
printf '%s\n' "$url" >> "$CURL_LOG"
case "$url" in */checksums.txt) cp "$CHECKSUMS" "$out" ;; *) cp "$ARCHIVE" "$out" ;; esac
`
	if err := os.WriteFile(filepath.Join(fakeBin, "curl"), []byte(fakeCurl), 0o755); err != nil {
		t.Fatal(err)
	}
	installDir, logPath := filepath.Join(tmp, "install"), filepath.Join(tmp, "curl.log")
	cmd := exec.Command("sh", filepath.Join("..", "install.sh"))
	cmd.Env = append(os.Environ(), "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"), "STRUCTLINT_VERSION=v0.6.0", "STRUCTLINT_INSTALL_DIR="+installDir, "ARCHIVE="+archive, "CHECKSUMS="+checksums, "CURL_LOG="+logPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install: %v\n%s", err, output)
	}
	output, err := exec.Command(filepath.Join(installDir, "structlint")).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "installed structlint" {
		t.Fatalf("binary: %v %q", err, output)
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "/releases/download/v0.6.0/"+asset) {
		t.Fatalf("wrong URL: %s", log)
	}
}
