//go:build mage

// Build tasks. Run with `go tool mage <target>`; `go tool mage -l` lists them.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	pkg      = "./cmd/savvy-go"
	outDir   = "dist"
	embedDir = "internal/webui/dist" // what the embed build tag compiles in
)

// Build groups the build targets.
type Build mg.Namespace

// Test groups the test targets.
type Test mg.Namespace

// Frontend builds the React app into public/build.
func (Build) Frontend() error {
	return sh.RunV(npm(), "run", "build")
}

// Dev builds a binary that serves the frontend from public/ on disk.
func (Build) Dev() error {
	return goBuild("dev", nil, os.Getenv("APP_ENV"))
}

// Release builds the frontend and a binary with it embedded. Honors
// APP_VERSION, APP_ENV (default production), GOOS and GOARCH.
func (Build) Release() error {
	mg.Deps(Build.Frontend)
	return Build{}.Embed()
}

// Embed is Release for an already built public/build.
func (Build) Embed() error {
	if err := stageFrontend(); err != nil {
		return err
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "production"
	}
	return goBuild("release", []string{"-tags", "embed", "-trimpath"}, env)
}

// Go runs the Go tests, with and without the embed tag.
func (Test) Go() error {
	if err := sh.RunV("go", "test", "./..."); err != nil {
		return err
	}
	if err := stageFrontend(); err != nil {
		return err
	}
	return sh.RunV("go", "test", "-tags", "embed", "./internal/webui/...", "./internal/httpserver/...")
}

// Frontend runs the frontend tests.
func (Test) Frontend() error {
	return sh.RunV(npm(), "test")
}

// All runs every test.
func (Test) All() {
	mg.SerialDeps(Test.Go, Test.Frontend)
}

// Clean removes build output.
func Clean() error {
	for _, p := range []string{outDir, embedDir, "public/build"} {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

// stageFrontend copies public/ (index.html, icons, build/) to where go:embed
// can reach it. It fails when the frontend is not built, so a release can
// never ship without it.
func stageFrontend() error {
	if _, err := os.Stat(filepath.Join("public", "build")); err != nil {
		return fmt.Errorf("public/build is missing, run `mage build:frontend`: %w", err)
	}
	if err := os.RemoveAll(embedDir); err != nil {
		return err
	}
	return os.CopyFS(embedDir, os.DirFS("public"))
}

func goBuild(name string, flags []string, appEnv string) error {
	bin := filepath.Join(outDir, "savvy-go")
	goos := firstNonEmpty(os.Getenv("GOOS"), runtime.GOOS)
	if goos == "windows" {
		bin += ".exe"
	}

	ldflags := "-X savvy-go/internal/version.Value=" + firstNonEmpty(os.Getenv("APP_VERSION"), "dev")
	if appEnv != "" {
		ldflags += " -X savvy-go/internal/version.Env=" + appEnv
	}
	if name == "release" {
		ldflags = "-s -w " + ldflags
	}

	args := append([]string{"build"}, flags...)
	args = append(args, "-ldflags", ldflags, "-o", bin, pkg)
	return sh.RunWithV(map[string]string{"CGO_ENABLED": "0"}, "go", args...)
}

func npm() string {
	if runtime.GOOS == "windows" {
		return "npm.cmd"
	}
	return "npm"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
