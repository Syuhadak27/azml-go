package ext

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"azml-go/config"
)

// runCmd executes a command and returns its trimmed stdout, or error
func runCmd(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// UpdateAndRestart performs the complete update.py workflow in native Go:
// 1. Checks UPSTREAM_REPO / git remote origin
// 2. Fetches and pulls the latest code
// 3. Compiles the Go binary if changes exist
// 4. Writes .restartmsg
// 5. Seamlessly executes/restarts the new binary without needing Docker rebuilds
func UpdateAndRestart(ctx context.Context, onStatus func(status string), recordRestartMsg func() error) error {
	cfg := config.Get()

	repo := strings.TrimSpace(cfg.UpstreamRepo)
	if repo == "" {
		repo = strings.TrimSpace(os.Getenv("UPSTREAM_REPO"))
	}

	branch := strings.TrimSpace(cfg.UpstreamBranch)
	if branch == "" {
		branch = strings.TrimSpace(os.Getenv("UPSTREAM_BRANCH"))
	}
	if branch == "" {
		branch = "main"
	}

	gitPath, err := exec.LookPath("git")
	hasGit := err == nil && gitPath != ""

	// If no explicit UPSTREAM_REPO, check if current git directory has origin
	if repo == "" && hasGit {
		if origin, oErr := runCmd(ctx, "", "git", "config", "--get", "remote.origin.url"); oErr == nil && origin != "" {
			repo = origin
		}
	}

	workDir, _ := os.Getwd()
	updated := false

	if repo != "" && hasGit {
		gitDir := filepath.Join(workDir, ".git")
		if _, statErr := os.Stat(gitDir); os.IsNotExist(statErr) {
			onStatus(fmt.Sprintf("• <i>Menyiapkan Git dan mengkloning <code>%s</code> (%s)...</i>", repo, branch))
			_, _ = runCmd(ctx, workDir, "git", "init")
			_, _ = runCmd(ctx, workDir, "git", "config", "user.email", "bot@azml.local")
			_, _ = runCmd(ctx, workDir, "git", "config", "user.name", "AZML-Go")
			_, _ = runCmd(ctx, workDir, "git", "remote", "add", "origin", repo)
			if _, fErr := runCmd(ctx, workDir, "git", "fetch", "origin", branch); fErr == nil {
				if _, cErr := runCmd(ctx, workDir, "git", "checkout", "-B", branch, "origin/"+branch); cErr == nil {
					updated = true
				}
			}
		} else {
			onStatus(fmt.Sprintf("• <i>Memeriksa pembaruan dari <code>%s</code> (%s)...</i>", repo, branch))
			_, _ = runCmd(ctx, workDir, "git", "remote", "set-url", "origin", repo)
			if _, fErr := runCmd(ctx, workDir, "git", "fetch", "origin", branch); fErr != nil {
				log.Printf("[Updater] git fetch error: %v", fErr)
				onStatus(fmt.Sprintf("⚠️ <i>Peringatan fetch: %v. Melanjutkan mulai ulang...</i>", fErr))
			} else {
				localHash, _ := runCmd(ctx, workDir, "git", "rev-parse", "HEAD")
				remoteHash, _ := runCmd(ctx, workDir, "git", "rev-parse", "FETCH_HEAD")

				if localHash != "" && remoteHash != "" && localHash != remoteHash {
					onStatus(fmt.Sprintf("• <i>Commit baru terdeteksi (<code>%.7s</code> ➔ <code>%.7s</code>). Menarik kode terbaru...</i>", localHash, remoteHash))
					if _, rErr := runCmd(ctx, workDir, "git", "reset", "--hard", "FETCH_HEAD"); rErr == nil {
						updated = true
					} else {
						onStatus(fmt.Sprintf("⚠️ <i>Git reset gagal: %v</i>", rErr))
					}
				} else {
					onStatus("• <i>Kode sudah mutakhir dengan commit terbaru.</i>")
				}
			}
		}
	} else if repo == "" {
		onStatus("• <i>UPSTREAM_REPO tidak dikonfigurasi. Melakukan mulai ulang biner secara langsung...</i>")
	} else {
		onStatus("⚠️ <i>Git tidak ditemukan di PATH. Melewati penarikan kode dan memulai ulang biner saat ini...</i>")
	}

	exePath, err := os.Executable()
	if err != nil || exePath == "" {
		exePath = "azml-go"
	}

	// If code was updated, compile with Go!
	if updated {
		goPath, gErr := exec.LookPath("go")
		if gErr != nil || goPath == "" {
			onStatus("⚠️ <i>Kompiler Go tidak ditemukan di PATH. Kode ditarik tetapi tidak dapat dikompilasi. Memulai ulang biner yang ada...</i>")
		} else {
			onStatus("• <i>Mengompilasi biner Go terbaru (<code>go build -ldflags=\"-w -s\"</code>)...</i>")
			buildCmd := exec.CommandContext(ctx, "go", "build", "-ldflags=-w -s", "-o", exePath, ".")
			buildCmd.Dir = workDir
			var buildOut bytes.Buffer
			buildCmd.Stdout = &buildOut
			buildCmd.Stderr = &buildOut

			if bErr := buildCmd.Run(); bErr != nil {
				log.Printf("[Updater] Build failed: %v\nOutput: %s", bErr, buildOut.String())
				onStatus(fmt.Sprintf("❌ <b>Kesalahan Kompilasi:</b>\n<pre>%s</pre>\n<i>Mulai ulang dibatalkan untuk menjaga bot tetap online.</i>", buildOut.String()))
				return fmt.Errorf("kompilasi gagal: %v", bErr)
			}
			onStatus("✅ <b>Kompilasi berhasil!</b> <i>Memulai ulang proses dengan kode terbaru...</i>")
		}
	}

	// Record restart message
	if recordRestartMsg != nil {
		_ = recordRestartMsg()
	}

	time.Sleep(800 * time.Millisecond)

	log.Printf("[Updater] Initiating process restart for %s...", exePath)
	if rErr := RestartProcess(exePath, os.Args, os.Environ()); rErr != nil {
		log.Printf("[Updater] RestartProcess error: %v, falling back to os.Exit(0)", rErr)
		os.Exit(0)
	}

	return nil
}
