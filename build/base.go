package build

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/autonomouskoi/mageutil"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	ArchAMD64 = "amd64"
	ArchARM64 = "arm64"
)

type Base struct {
	BaseDir        string
	DistDir        string
	WorkDir        string
	ReleaseVersion string
	MainPath       string
	RunPath        string
	GoARCH         string
	GoOS           string
	ExecPath       string
	ExtraEnv       map[string]string
	LDFlags        string
}

func NewBase() (*Base, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}
	b := &Base{BaseDir: baseDir}
	b.WorkDir = filepath.Join(baseDir, "work")
	b.DistDir = filepath.Join(b.WorkDir, "dist")
	b.LDFlags = "-s -w"

	return b, nil
}

func (b *Base) releaseVersion() error {
	versionB, err := os.ReadFile(filepath.Join(b.BaseDir, "VERSION"))
	if err != nil {
		return fmt.Errorf("reading VERSION: %w", err)
	}
	b.ReleaseVersion = "v" + strings.TrimSpace(string(versionB))
	return nil
}

func (b *Base) ReleaseDeps() error {
	mg.Deps(b.releaseVersion)
	b.DistDir = filepath.Join(b.WorkDir, "dist")
	/*
		if err := sh.Rm(b.DistDir); err != nil {
			return fmt.Errorf("removing %s: %w", b.DistDir, err)
		}
	*/
	if err := mageutil.Mkdir(b.DistDir); err != nil {
		return fmt.Errorf("creating %s: %w", b.DistDir, err)
	}
	b.MainPath = filepath.Join(b.WorkDir, "app", "ak")
	b.RunPath = filepath.Join(b.BaseDir, "run")
	return nil
}

func (b *Base) MageBuild(target string, path ...string) error {
	buildPath := filepath.Join(append([]string{b.WorkDir}, path...)...)
	return sh.Run("mage", "-d", buildPath, target)
}

func (b *Base) workDir() error {
	return mageutil.Mkdir(b.WorkDir)
}

func (b *Base) preReqs() error {
	return mageutil.HasExec("git", "npm", "protoc")
}

func (b *Base) cloneRepos() error {
	mg.Deps(b.workDir, b.preReqs)
	if err := os.Chdir(b.WorkDir); err != nil {
		return fmt.Errorf("switching to %s: %w", b.WorkDir, err)
	}
	for _, repo := range []string{"akcore", "trackstar", "twitch"} {
		outPath := filepath.Join(b.WorkDir, repo)
		if _, err := os.Stat(outPath); err == nil {
			continue
		}
		repo = "https://github.com/AutonomousKoi/" + repo
		if err := sh.Run("git", "clone", "--depth", "1", repo); err != nil {
			return fmt.Errorf("cloning %s: %w", repo, err)
		}
	}
	return nil
}

func (b *Base) copyApp() error {
	mg.Deps(b.workDir)
	destDir := filepath.Join(b.WorkDir, "app")
	srcDir := filepath.Join(b.BaseDir, "cmd")
	if err := mageutil.CopyRecursively(destDir, srcDir); err != nil {
		return fmt.Errorf("copying %s -> %s: %w", srcDir, destDir, err)
	}
	if err := mageutil.CopyInDir(destDir, b.BaseDir, "go.mod", "go.sum"); err != nil {
		return fmt.Errorf("copying mod files: %w", err)
	}
	return nil
}

func (b *Base) npmInstall() error {
	mg.Deps(b.cloneRepos, b.preReqs)
	for _, path := range []string{
		filepath.Join("akcore", "web", "content"),
		"trackstar",
		filepath.Join("trackstar", "stagelinq"),
		"twitch",
	} {
		contentPath := filepath.Join(b.WorkDir, path)
		nmPath := filepath.Join(contentPath, "node_modules")
		if _, err := os.Stat(nmPath); err == nil {
			continue
		}
		if err := os.Chdir(contentPath); err != nil {
			return fmt.Errorf("switching to %s: %w", contentPath, err)
		}
		if err := sh.Run("npm", "install"); err != nil {
			return fmt.Errorf("running npm install: %w", err)
		}
	}
	return nil
}

func (b *Base) goWorkspace() error {
	mg.Deps(b.workDir, b.cloneRepos, b.copyApp)
	goWorkPath := filepath.Join(b.WorkDir, "go.work")
	if _, err := os.Stat(goWorkPath); err == nil {
		return nil
	}
	if err := os.Chdir(b.WorkDir); err != nil {
		return fmt.Errorf("switching to %s: %w", b.WorkDir, err)
	}
	if err := sh.Run("go", "work", "init"); err != nil {
		return fmt.Errorf("initializing Go workspace: %w", err)
	}
	err := sh.Run("go", "work", "use", "akcore", "app", "trackstar", "twitch")
	if err != nil {
		return fmt.Errorf("configuring Go workspace: %w", err)
	}
	return nil
}

func (b *Base) buildPre() {
	mg.SerialDeps(b.goWorkspace, b.npmInstall)
}

func (b *Base) coreDeps() error {
	mg.Deps(b.buildPre)
	return b.MageBuild("dev", "akcore", "build")
}

func (b *Base) modBuild() error {
	mg.Deps(b.coreDeps)
	for _, mod := range []string{"twitch", "trackstar"} {
		buildPath := filepath.Join(mod, "build")
		if err := b.MageBuild("all", buildPath); err != nil {
			return fmt.Errorf("building %s: %w", mod, err)
		}
	}
	if err := b.MageBuild("webzip", "akcore", "build"); err != nil {
		return fmt.Errorf("building akcore webzip: %w", err)
	}
	return nil
}

func (b *Base) compile() error {
	mg.Deps(b.ReleaseDeps, b.modBuild)

	env := map[string]string{
		"GOOS":        b.GoOS,
		"GOARCH":      b.GoARCH,
		"CGO_ENABLED": "1",
	}
	maps.Insert(env, maps.All(b.ExtraEnv))
	return sh.RunWith(env,
		"go", "build",
		"-o", b.ExecPath,
		"-trimpath",
		"-ldflags", b.LDFlags+" -X github.com/autonomouskoi/akcore.Version="+b.ReleaseVersion[1:],
		b.MainPath,
	)
}

func (b *Base) preRelease() error {
	b.DistDir = filepath.Join(b.DistDir, b.GoOS, b.GoARCH)
	return os.MkdirAll(b.DistDir, 0o755)
}
