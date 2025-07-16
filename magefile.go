//go:build mage
// +build mage

package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/autonomouskoi/mageutil"
)

var (
	baseDir        string
	distDir        string
	workDir        string
	akCoreDir      string
	releaseVersion string
	mainPath       string
	runPath        string
)

func init() {
	var err error
	baseDir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	akCoreDir = filepath.Join(workDir, "akcore")
	workDir = filepath.Join(baseDir, "work")
}

func Clean() error {
	return sh.Rm(workDir)
}

func WorkDir() error {
	return mageutil.Mkdir(workDir)
}

func Prereqs() error {
	return mageutil.HasExec("git", "npm", "protoc")
}

func CloneRepos() error {
	mg.Deps(WorkDir, Prereqs)
	if err := os.Chdir(workDir); err != nil {
		return fmt.Errorf("switching to %s: %w", workDir, err)
	}
	for _, repo := range []string{"akcore", "trackstar", "twitch"} {
		outPath := filepath.Join(workDir, repo)
		if _, err := os.Stat(outPath); err == nil {
			continue
		}
		repo = "https://github.com/AutonomousKoi/" + repo
		if err := sh.Run("git", "clone", repo); err != nil {
			return fmt.Errorf("cloning %s: %w", repo, err)
		}
	}
	return nil
}

func CopyApp() error {
	mg.Deps(WorkDir)
	destDir := filepath.Join(workDir, "app")
	srcDir := filepath.Join(baseDir, "cmd")
	if err := mageutil.CopyRecursively(destDir, srcDir); err != nil {
		return fmt.Errorf("copying %s -> %s: %w", srcDir, destDir, err)
	}
	if err := mageutil.CopyInDir(destDir, baseDir, "go.mod", "go.sum"); err != nil {
		return fmt.Errorf("copying mod files: %w", err)
	}
	return nil
}

func NPMInstall() error {
	mg.Deps(CloneRepos, Prereqs)
	for _, path := range []string{
		filepath.Join("akcore", "web", "content"),
		"trackstar",
		filepath.Join("trackstar", "stagelinq"),
		"twitch",
	} {
		contentPath := filepath.Join(workDir, path)
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

func GoWorkspace() error {
	mg.Deps(WorkDir, CloneRepos, CopyApp)
	goWorkPath := filepath.Join(workDir, "go.work")
	if _, err := os.Stat(goWorkPath); err == nil {
		return nil
	}
	if err := os.Chdir(workDir); err != nil {
		return fmt.Errorf("switching to %s: %w", workDir, err)
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

func mageBuild(target string, path ...string) error {
	buildPath := filepath.Join(append([]string{workDir}, path...)...)
	return sh.Run("mage", "-d", buildPath, target)
}

func BuildPre() {
	mg.SerialDeps(GoWorkspace, NPMInstall)
}

func CoreDeps() error {
	mg.Deps(BuildPre)
	return mageBuild("dev", "akcore", "build")
}

func Build() error {
	mg.Deps(CoreDeps)
	for _, mod := range []string{"twitch", "trackstar"} {
		buildPath := filepath.Join(mod, "build")
		if err := mageBuild("all", buildPath); err != nil {
			return fmt.Errorf("building %s: %w", mod, err)
		}
	}
	if err := mageBuild("webzip", "akcore", "build"); err != nil {
		return fmt.Errorf("building akcore webzip: %w", err)
	}
	return nil
}

func ReleaseDeps() error {
	versionB, err := os.ReadFile(filepath.Join(baseDir, "VERSION"))
	if err != nil {
		return fmt.Errorf("reading VERSION: %w", err)
	}
	releaseVersion = "v" + strings.TrimSpace(string(versionB))
	distDir = filepath.Join(workDir, "dist")
	if err := sh.Rm(distDir); err != nil {
		return fmt.Errorf("removing %s: %w", distDir, err)
	}
	if err := mageutil.Mkdir(distDir); err != nil {
		return fmt.Errorf("creating %s: %w", distDir, err)
	}
	mainPath = filepath.Join(workDir, "app", "ak")
	runPath = filepath.Join(baseDir, "run")
	return nil
}

func ReleaseMac() error {
	mg.Deps(ReleaseDeps, Build)
	baseName := "ak-mac-" + releaseVersion
	outPath := filepath.Join(distDir, baseName)
	err := sh.RunWith(map[string]string{},
		"go", "build",
		"-o", outPath,
		"-ldflags", "-s -w -X github.com/autonomouskoi/akcore.Version="+releaseVersion[1:],
		mainPath,
	)
	if err != nil {
		return fmt.Errorf("building %s: %w", outPath, err)
	}

	dmgTmplPath := filepath.Join(runPath, "AK-tmpl.dmg.gz")
	tmplFH, err := os.Open(dmgTmplPath)
	if err != nil {
		return fmt.Errorf("opening DMG template %s: %w", dmgTmplPath, err)
	}
	defer tmplFH.Close()
	gzR, err := gzip.NewReader(tmplFH)
	if err != nil {
		return fmt.Errorf("creating DMG template decompressor: %w", err)
	}
	dmgFilePath := filepath.Join(distDir, "AK-tmpl.dmg")
	dmgFH, err := os.Create(dmgFilePath)
	if err != nil {
		return fmt.Errorf("creating DMG file %s: %w", dmgFilePath, err)
	}
	defer dmgFH.Close()
	if _, err := io.Copy(dmgFH, gzR); err != nil {
		return fmt.Errorf("decompressing DMG file: %w", err)
	}
	if err := dmgFH.Sync(); err != nil {
		return fmt.Errorf("syncing DMG file: %w", err)
	}

	// resize to hold the executable + 1MB overhead
	stat, err := os.Stat(outPath)
	if err != nil {
		return fmt.Errorf("statting executable: %w", err)
	}
	err = sh.Run("hdiutil", "resize", "-size", strconv.Itoa(int(stat.Size())+(64*1024*1024)), dmgFilePath)
	if err != nil {
		return fmt.Errorf("resizing DMG file: %w", err)
	}

	// mount the dmg
	appDir := filepath.Join(distDir, "mac")
	if err := mageutil.Mkdir(appDir); err != nil {
		return fmt.Errorf("creating app dir: %w", err)
	}
	err = sh.Run("hdiutil", "attach", dmgFilePath, "-noautoopen", "-mountpoint", appDir)
	if err != nil {
		return fmt.Errorf("attaching DMG: %w", err)
	}
	detached := false
	defer func() {
		if !detached {
			sh.Run("hdiutil", "detach", appDir+"/")
		}
	}()

	// copy stuff
	appExecPath := filepath.Join(appDir, "AutonomousKoi.app", "Contents", "MacOS", "ak")
	if err := sh.Copy(appExecPath, outPath); err != nil {
		return fmt.Errorf("copying app executable: %w", err)
	}
	if err := os.Chmod(appExecPath, 0555); err != nil {
		return fmt.Errorf("setting app executable permissions: %w", err)
	}
	licDestPath := filepath.Join(appDir, "LICENSE")
	licSrcPath := filepath.Join(baseDir, "LICENSE")
	if err := sh.Copy(licDestPath, licSrcPath); err != nil {
		return fmt.Errorf("copying LICENSE: %w", err)
	}
	err = sh.Run("/bin/sh",
		filepath.Join(baseDir, "third_party_libs_tool"),
		filepath.Join(appDir, "AutonomousKoi.app"),
	)
	if err != nil {
		return fmt.Errorf("linking libs: %w", err)
	}
	// signing
	err = sh.Run("codesign",
		"-s", "D00E79F3D70A4981BC28490E48E91E7430B4A245",
		"--timestamp",
		"-o", "runtime",
		filepath.Join(appDir, "AutonomousKoi.app", "Contents", "Frameworks", "libcrypto.3.dylib"),
	)
	if err != nil {
		return fmt.Errorf("signing libcrypto: %w", err)
	}
	err = sh.Run("codesign",
		"-s", "D00E79F3D70A4981BC28490E48E91E7430B4A245",
		"--entitlements", filepath.Join(baseDir, "ak.entitlements"),
		"--timestamp",
		"-o", "runtime",
		appExecPath,
	)
	if err != nil {
		return fmt.Errorf("signing ak: %w", err)
	}

	// detach, compress
	if err := sh.Run("hdiutil", "detach", appDir+"/"); err != nil {
		return fmt.Errorf("detaching DMG %s: %w", appDir, err)
	}
	detached = true
	err = sh.Run("hdiutil", "convert", dmgFilePath,
		"-format", "UDZO",
		"-imagekey", "zlib-level=9",
		"-o", filepath.Join(distDir, "AutonomousKoi-mac-"+releaseVersion+".dmg"),
	)
	if err != nil {
		return fmt.Errorf("compressing DMG: %w", err)
	}

	return nil
}

var goarch string

func ReleaseLinuxAMD64() {
	goarch = "amd64"
	mg.Deps(releaseLinux)
}

func ReleaseLinuxARM64() {
	goarch = "arm64"
	mg.Deps(releaseLinux)
}

func releaseLinux() error {
	mg.Deps(ReleaseDeps, Build)

	thisPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	defer os.Chdir(thisPath)
	if err := os.Chdir(mainPath); err != nil {
		return fmt.Errorf("switching to main source dir: %w", err)
	}

	exeName := "autonomouskoi"
	outPath := filepath.Join(distDir, exeName)
	err = sh.RunWith(map[string]string{
		"GOARCH": goarch,
	},
		"go", "build",
		"-o", outPath,
		"-ldflags", "-s -w -X github.com/autonomouskoi/akcore.Version="+releaseVersion[1:],
		mainPath,
	)
	if err != nil {
		return fmt.Errorf("building %s: %w", outPath, err)
	}
	zipPath := filepath.Join(distDir, "AutonomousKoi-linux-"+goarch+"-"+releaseVersion+".zip")
	err = mageutil.ZipFiles(zipPath, map[string]string{
		filepath.Join(baseDir, "LICENSE"): "LICENSE",
		outPath:                           exeName,
	})
	return nil
}

func ReleaseWin() error {
	mg.Deps(ReleaseDeps, Build)

	thisPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	defer os.Chdir(thisPath)
	if err := os.Chdir(mainPath); err != nil {
		return fmt.Errorf("switching to main source dir: %w", err)
	}

	exeName := "ak.exe"
	outPath := filepath.Join(distDir, exeName)
	err = sh.RunWith(map[string]string{
		"CGO_ENABLED": "1",
		"CGO_CFLAGS":  "-I/mingw64/include",
		"MSYSTEM":     "MINGW64",
	},
		"go", "build",
		"-o", outPath,
		"-ldflags", "-H=windowsgui -X github.com/autonomouskoi/akcore.Version="+releaseVersion[1:],
		mainPath,
	)
	if err != nil {
		return fmt.Errorf("building %s: %w", outPath, err)
	}
	libCryptoDll := "libcrypto-3-x64.dll"
	libCryptoSrc := filepath.Join(`C:\`, "msys64", "mingw64", "bin", libCryptoDll)
	zipPath := filepath.Join(distDir, "AutonomousKoi-win-"+releaseVersion+".zip")
	err = mageutil.ZipFiles(zipPath, map[string]string{
		filepath.Join(baseDir, "LICENSE"): "LICENSE",
		outPath:                           exeName,
		libCryptoSrc:                      libCryptoDll,
	})
	return err
}

var Default = Build
