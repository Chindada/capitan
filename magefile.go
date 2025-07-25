//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	plstformLinux   = "linux"
	platformWindows = "windows"
	platformDarwin  = "darwin"

	archAmd64 = "amd64"
	archArm64 = "arm64"
	archArm   = "arm"
)

const (
	cgoEnable = "0"
)

const (
	armVersion7 = "7"
)

const (
	cmdDir    = "cmd"
	outDir    = "bin"
	outPrefix = ""
)

const (
	buildTagDebug = "debug"
	buildTagProd  = "prod"
)

var (
	freshInstall bool
	armVersion   string
	buildTag     string = buildTagDebug
	arch         string = runtime.GOARCH
	platform     string = runtime.GOOS
)

var Aliases = map[string]any{
	"prod":     Platform.Prod,
	"amd64":    Platform.Amd64,
	"arm64":    Platform.Arm64,
	"arm32":    Platform.Arm32,
	"linux":    Platform.Linux,
	"win":      Platform.Windows,
	"darwin":   Platform.Darwin,
	"lint":     Lint.Lint,
	"lintcc":   Lint.CleanLintCache,
	"init-db":  Db.Init,
	"start-db": Db.Start,
	"stop-db":  Db.Stop,
	"test":     Coverage,
	"update":   GoModUpdate,
	"fi":       Dep.FreshInstall,
}

type Platform mg.Namespace

// Set build tag to debug
func (Platform) Prod() {
	buildTag = buildTagProd
}

// Set architecture to amd64
func (Platform) Amd64() {
	arch = archAmd64
}

// Set architecture to arm64
func (Platform) Arm64() {
	arch = archArm64
}

// Set architecture to arm32
func (Platform) Arm32() {
	arch = archArm
	armVersion = armVersion7
}

// Set platform to linux
func (Platform) Linux() {
	platform = plstformLinux
}

// Set platform to windows
func (Platform) Windows() {
	arch = archAmd64
	platform = platformWindows
}

// Set platform to darwin
func (Platform) Darwin() {
	platform = platformDarwin
}

type Dep mg.Namespace

func (Dep) FreshInstall() {
	freshInstall = true
}

// Set GOPRIVATE to github.com/chindada/*
func (Dep) SetGoPrivate() error {
	fmt.Println("Setting GOPRIVATE...")
	return sh.RunV("go", "env", "-w", "GOPRIVATE=github.com/chindada/*")
}

func checkAndInstall(name, installCmd string, args ...string) error {
	err := sh.Run("which", name)
	if err != nil || freshInstall {
		fmt.Printf("Installing %s...\n", name)
		return sh.RunV(installCmd, args...)
	}
	return nil
}

// Install go.uber.org/mock/mockgen@latest
func (Dep) InstallMockgen() error {
	return checkAndInstall("mockgen", "go", "install", "go.uber.org/mock/mockgen@latest")
}

// Install github.com/swaggo/swag/cmd/swag@latest
func (Dep) InstallSwag() error {
	return checkAndInstall("swag", "go", "install", "github.com/swaggo/swag/cmd/swag@latest")
}

// Install go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
func (Dep) InstallGolangciLint() error {
	return checkAndInstall(
		"golangci-lint",
		"go",
		"install",
		"github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest",
	)
}

// Install @redocly/cli@latest
func (Dep) InstallRedocly() error {
	return checkAndInstall("redocly", "npm", "install", "-g", "@redocly/cli@latest")
}

// Remove mocks
func (Dep) RemoveMocks() error {
	fmt.Println("Removing Mocks...")
	mocksDir := []string{
		"internal/usecases/mocks",
		"internal/usecases/repo/mocks",
	}
	for _, dir := range mocksDir {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}

type Gen mg.Namespace

// Run go generate ./...
func (Gen) GoGenerate() error {
	mg.Deps(Dep.RemoveMocks)
	fmt.Println("Running go generate...")
	return sh.RunV("go", "generate", "./...")
}

// Run scripts/generate_swagger.sh
func (Gen) SwagGenerate() error {
	fmt.Println("Generating swagger docs...")
	return sh.RunV("sh", "./scripts/generate_swagger.sh")
}

type Lint mg.Namespace

// Clean lint cache
func (Lint) CleanLintCache() error {
	fmt.Println("Cleaning lint cache...")
	return sh.RunV("golangci-lint", "cache", "clean")
}

// Run Lint project
func (Lint) Lint() error {
	mg.SerialDeps(Dep.InstallGolangciLint)
	fmt.Println("Linting...")
	return sh.RunV("golangci-lint", "run")
}

type Test mg.Namespace

// Install gotestsum
func (Test) InstallGotestsum() error {
	return checkAndInstall("gotestsum", "go", "install", "gotest.tools/gotestsum@latest")
}

// Test Run gotestsum --junitfile report.xml --format testname -- -coverprofile=coverage.txt ./...
func (Test) Run() error {
	mg.Deps(Test.InstallGotestsum)
	fmt.Println("Running tests...")
	return sh.RunV(
		"gotestsum",
		"--junitfile",
		"report.xml",
		"--format",
		"testname",
		"--",
		"-coverprofile=coverage.txt",
		"./...",
	)
}

type Db mg.Namespace

func (Db) CheckDbToolExist() error {
	if _, err := os.Stat("./bin/dbtool"); os.IsNotExist(err) {
		return fmt.Errorf("dbtool not found in ./bin, please run 'mage build' first")
	}
	return nil
}

func (Db) Init() error {
	mg.Deps(Db.CheckDbToolExist)
	fmt.Println("Initializing database...")
	err := sh.RunV("./bin/dbtool", "init", "-f", "-s", "--db-name", "capitan")
	if err != nil {
		return err
	}
	return sh.RunV("./bin/dbtool", "migrate", "up", "--db-name", "capitan")
}

func (Db) Start() error {
	mg.Deps(Db.CheckDbToolExist)
	fmt.Println("Starting database...")
	return sh.RunV("./bin/dbtool", "start", "--db-name", "capitan")
}

func (Db) Stop() error {
	mg.Deps(Db.CheckDbToolExist)
	fmt.Println("Stopping database...")
	return sh.RunV("./bin/dbtool", "stop", "--db-name", "capitan")
}

// Check coverage
func Coverage() error {
	mg.SerialDeps(Dep.SetGoPrivate, Lint.Lint, Test.Run)
	fmt.Println("Checking coverage...")
	return sh.RunV("go", "tool", "cover", "-func", "coverage.txt")
}

// Build -
func Build() error {
	mg.Deps(Dep.SetGoPrivate)
	if buildTag == buildTagDebug {
		mg.Deps(Dep.InstallMockgen, Dep.InstallSwag, Dep.InstallRedocly)
		mg.Deps(Gen.GoGenerate, Gen.SwagGenerate)
	}
	paths, err := os.ReadDir(cmdDir)
	if err != nil {
		return err
	}
	envVar := map[string]string{
		"CGO_ENABLED": cgoEnable,
		"GOOS":        platform,
		"GOARCH":      arch,
		"GOARM":       armVersion,
	}

	var wg sync.WaitGroup
	// A channel to collect errors from goroutines.
	errCh := make(chan error, len(paths))

	for _, dir := range paths {
		if !dir.IsDir() {
			continue
		}
		wg.Add(1)
		go func(dirName string) {
			defer wg.Done()
			ldflags := "-s -w"
			input := fmt.Sprintf("./%s/%s", cmdDir, dirName)
			outputName := fmt.Sprintf("%s/%s%s", outDir, outPrefix, dirName)
			if platform == platformWindows {
				outputName = fmt.Sprintf("%s.exe", outputName)
			}
			fmt.Printf("Building %s %s for %s %s\n", buildTag, input, platform, arch)
			if err := sh.RunWithV(
				envVar, "go", "build", fmt.Sprintf("-ldflags=%s", ldflags), "-tags", buildTag, "-o", outputName, input,
			); err != nil {
				errCh <- err
			}
		}(dir.Name())
	}

	// Wait for all builds to complete and close the error channel.
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Check for any errors that occurred during the build.
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// Run scripts/gomod_update.sh
func GoModUpdate() error {
	mg.Deps(Dep.SetGoPrivate)
	fmt.Println("Go mod update...")
	return sh.RunV("sh", "./scripts/gomod_update.sh")
}

// Clean output directory
func Clean() error {
	fmt.Println("Cleaning...")
	return os.RemoveAll(outDir)
}
