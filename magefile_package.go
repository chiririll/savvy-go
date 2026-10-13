//go:build mage

package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Package groups the Linux package targets. They need nfpm on the PATH,
// APP_VERSION and the Linux binary in dist/savvy-go (build:release with
// TARGET_GOOS=linux).
type Package mg.Namespace

type format struct {
	name string
	user string // service account, differs between the distributions
	glob string // what nfpm names the file
}

var (
	deb = format{"deb", "www-data", "savvy-go_*.deb"}
	rpm = format{"rpm", "savvy-go", "savvy-go-*.rpm"}
)

// Deb builds dist/savvy-go.deb.
func (Package) Deb() error { return pack(deb) }

// Rpm builds dist/savvy-go.rpm.
func (Package) Rpm() error { return pack(rpm) }

func pack(f format) error {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		return errors.New("APP_VERSION is required")
	}
	if _, err := os.Stat(filepath.Join(outDir, "savvy-go")); err != nil {
		return errors.New("dist/savvy-go is missing, run `go tool mage build:release` for linux first")
	}
	if _, err := exec.LookPath("nfpm"); err != nil {
		return errors.New("nfpm is not installed, see https://nfpm.goreleaser.com")
	}
	out, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}

	// nfpm does not expand env vars in content sources or file owners, so the
	// unit and the config are rendered with the account name substituted.
	stage := filepath.Join(out, "stage")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for _, name := range []string{"savvy-go.service", "nfpm.yaml"} {
		raw, err := os.ReadFile(filepath.Join("deploy", "nfpm", name))
		if err != nil {
			return err
		}
		rendered := strings.ReplaceAll(string(raw), "@APP_USER@", f.user)
		if err := os.WriteFile(filepath.Join(stage, name), []byte(rendered), 0o644); err != nil {
			return err
		}
	}

	old, _ := filepath.Glob(filepath.Join(out, f.glob))
	for _, p := range old {
		_ = os.Remove(p)
	}

	cmd := exec.Command("nfpm", "package", "--config", filepath.Join(stage, "nfpm.yaml"), "--packager", f.name, "--target", out)
	cmd.Dir = filepath.Join("deploy", "nfpm") // the content sources in nfpm.yaml are relative to it
	cmd.Env = append(os.Environ(), "APP_VERSION="+version)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nfpm: %w", err)
	}

	built, _ := filepath.Glob(filepath.Join(out, f.glob))
	if len(built) != 1 {
		return fmt.Errorf("expected one %s, found %v", f.glob, built)
	}
	final := filepath.Join(out, "savvy-go."+f.name)
	if err := copyFile(built[0], final); err != nil {
		return err
	}
	fmt.Printf("Wrote %s and %s (VERSION=%s)\n", built[0], final, version)
	return nil
}

// Release builds everything a GitHub release ships into dist/: Linux and
// Windows binaries with the frontend embedded, the deb and rpm packages and
// SHA256SUMS. Needs APP_VERSION and nfpm.
func Release() error {
	if os.Getenv("APP_VERSION") == "" {
		return errors.New("APP_VERSION is required")
	}
	mg.Deps(Build.Frontend)
	if err := stageFrontend(); err != nil {
		return err
	}

	env := firstNonEmpty(os.Getenv("APP_ENV"), "production")
	linux := filepath.Join(outDir, "savvy-go") // what the packages are made from
	if err := compile(buildOpts{goos: "linux", goarch: "amd64", out: linux, embed: true, env: env}); err != nil {
		return err
	}
	windows := filepath.Join(outDir, "savvy-go-windows-amd64.exe")
	if err := compile(buildOpts{goos: "windows", goarch: "amd64", out: windows, embed: true, env: env}); err != nil {
		return err
	}
	linuxNamed := filepath.Join(outDir, "savvy-go-linux-amd64")
	if err := copyFile(linux, linuxNamed); err != nil {
		return err
	}

	if err := pack(deb); err != nil {
		return err
	}
	if err := pack(rpm); err != nil {
		return err
	}
	return checksums(linuxNamed, windows, filepath.Join(outDir, "savvy-go.deb"), filepath.Join(outDir, "savvy-go.rpm"))
}

// checksums writes dist/SHA256SUMS in the format `sha256sum -c` reads.
func checksums(files ...string) error {
	sort.Strings(files)
	var sums strings.Builder
	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return err
		}
		fmt.Fprintf(&sums, "%x  %s\n", h.Sum(nil), filepath.Base(name))
	}
	return os.WriteFile(filepath.Join(outDir, "SHA256SUMS"), []byte(sums.String()), 0o644)
}

// Packages builds the deb and rpm from the working tree, installs them in
// systemd containers and checks that the service survives install, a crash,
// an upgrade and removal. Needs Docker and Node. SKIP_FRONTEND=1 reuses an
// existing public/build.
func (Test) Packages() error { return testPackages(deb, rpm) }

// Deb is Packages for the deb only.
func (Test) Deb() error { return testPackages(deb) }

// Rpm is Packages for the rpm only.
func (Test) Rpm() error { return testPackages(rpm) }

func testPackages(formats ...format) error {
	version := firstNonEmpty(os.Getenv("APP_VERSION"), "0.0.0-dev")
	os.Setenv("APP_VERSION", version)

	fmt.Println("== build Go binary with the frontend embedded")
	if _, err := os.Stat(filepath.Join("public", "build")); os.Getenv("SKIP_FRONTEND") != "1" || err != nil {
		if err := (Build{}).Frontend(); err != nil {
			return err
		}
	}
	if err := stageFrontend(); err != nil {
		return err
	}
	if err := compile(buildOpts{goos: "linux", goarch: "amd64", embed: true, env: "production"}); err != nil {
		return err
	}

	// The packaging container has nfpm but no Go, so it gets mage compiled.
	fmt.Println("== package (in container)")
	runner := filepath.Join(outDir, "mage-linux")
	defer os.Remove(runner)
	if err := sh.Run("go", "tool", "mage", "-goos", "linux", "-goarch", "amd64", "-compile", runner); err != nil {
		return err
	}
	testDir := filepath.Join("deploy", "nfpm", "test")
	if err := sh.RunV("docker", "build", "-q", "-t", "savvy-go-packager", "-f", filepath.Join(testDir, "Dockerfile.package"), testDir); err != nil {
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	args := []string{"run", "--rm"}
	if os.Getuid() >= 0 { // keep dist/ owned by the current user instead of root
		args = append(args, "--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()))
	}
	var cmds []string
	for _, f := range formats {
		cmds = append(cmds, "./dist/mage-linux package:"+f.name)
	}
	args = append(args, "-v", root+":/src", "-e", "APP_VERSION="+version, "-e", "HOME=/tmp",
		"savvy-go-packager", "bash", "-c", strings.Join(cmds, " && "))
	if err := sh.RunV("docker", args...); err != nil {
		return err
	}

	for _, f := range formats {
		fmt.Printf("\n== test %s\n", f.name)
		if err := sh.RunV("node", filepath.Join(testDir, "test-package.mts"), f.name); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
