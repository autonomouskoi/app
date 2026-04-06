package build

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/autonomouskoi/mageutil"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Mac struct {
	*Base

	dmgPath       string
	mountPath     string
	mountDetached bool
}

func NewMac(b *Base) Mac {
	m := Mac{
		Base: b,
	}
	m.GoOS = "darwin"
	return m
}

func (m Mac) Release() error {
	mg.Deps(m.Base.releaseVersion)
	m.ExecPath = filepath.Join(m.DistDir, fmt.Sprintf("ak-mac-%s-%s", m.GoARCH, m.ReleaseVersion))
	mg.Deps(m.Base.compile)
	mg.Deps(m.Base.preRelease)

	if err := m.startImage(); err != nil {
		return fmt.Errorf("starting image: %w", err)
	}
	defer m.closeImage()

	// copying
	if err := m.copyImageFiles(); err != nil {
		return fmt.Errorf("copying image files: %w", err)
	}

	return m.compressImage()
}

func (m *Mac) startImage() error {
	dmgTmplPath := filepath.Join(m.RunPath, "AK-tmpl.dmg.gz")
	tmplFH, err := os.Open(dmgTmplPath)
	if err != nil {
		return fmt.Errorf("opening DMG template %s: %w", dmgTmplPath, err)
	}
	defer tmplFH.Close()
	gzR, err := gzip.NewReader(tmplFH)
	if err != nil {
		return fmt.Errorf("creating DMG template decompressor: %w", err)
	}
	m.dmgPath = filepath.Join(m.DistDir, "AK-tmpl.dmg")
	dmgFH, err := os.Create(m.dmgPath)
	if err != nil {
		return fmt.Errorf("creating DMG file %s: %w", m.dmgPath, err)
	}
	defer dmgFH.Close()
	if _, err := io.Copy(dmgFH, gzR); err != nil {
		return fmt.Errorf("decompressing DMG file: %w", err)
	}
	if err := dmgFH.Sync(); err != nil {
		return fmt.Errorf("syncing DMG file: %w", err)
	}

	// resize to hold the executable + 1MB overhead
	stat, err := os.Stat(m.ExecPath)
	if err != nil {
		return fmt.Errorf("statting executable: %w", err)
	}
	err = sh.Run("hdiutil", "resize", "-size", strconv.Itoa(int(stat.Size())+(64*1024*1024)), m.dmgPath)
	if err != nil {
		return fmt.Errorf("resizing DMG file: %w", err)
	}

	// mount the dmg
	m.mountPath = m.DistDir
	mageutil.VerboseF("mount path: %q\n", m.mountPath)
	if err := mageutil.Mkdir(m.mountPath); err != nil {
		return fmt.Errorf("creating app dir: %w", err)
	}
	err = sh.Run("hdiutil", "attach", m.dmgPath, "-noautoopen", "-mountpoint", m.mountPath)
	if err != nil {
		return fmt.Errorf("attaching DMG: %w", err)
	}
	return nil
}

func (m *Mac) closeImage() error {
	if m.mountDetached {
		return nil
	}
	if err := sh.Run("hdiutil", "detach", m.mountPath+"/"); err != nil {
		return fmt.Errorf("detaching DMG %s: %w", m.mountPath, err)
	}
	m.mountDetached = true
	return nil
}

func (m *Mac) compressImage() error {
	// detach, compress
	if err := m.closeImage(); err != nil {
		return err
	}
	err := sh.Run("hdiutil", "convert", m.dmgPath,
		"-format", "UDZO",
		"-imagekey", "zlib-level=9",
		"-o", filepath.Join(m.DistDir, "..", "..", fmt.Sprintf("AutonomousKoi-mac-%s-%s.dmg", m.GoARCH, m.ReleaseVersion)),
	)
	if err != nil {
		return fmt.Errorf("compressing DMG: %w", err)
	}
	return nil
}

func (m *Mac) copyImageFiles() error {
	// exec
	appExecPath := filepath.Join(m.DistDir, "AutonomousKoi.app", "Contents", "MacOS", "ak")
	if err := sh.Copy(appExecPath, m.ExecPath); err != nil {
		return fmt.Errorf("copying app executable: %w", err)
	}
	if err := os.Chmod(appExecPath, 0o555); err != nil {
		return fmt.Errorf("setting app executable permissions: %w", err)
	}
	// license
	licDestPath := filepath.Join(m.DistDir, "LICENSE")
	licSrcPath := filepath.Join(m.BaseDir, "LICENSE")
	if err := sh.Copy(licDestPath, licSrcPath); err != nil {
		return fmt.Errorf("copying LICENSE: %w", err)
	}
	// relink
	err := sh.Run("/bin/sh",
		filepath.Join(m.BaseDir, "third_party_libs_tool"),
		filepath.Join(m.DistDir, "AutonomousKoi.app"),
	)
	if err != nil {
		return fmt.Errorf("linking libs: %w", err)
	}
	// signing
	if m.GoARCH == ArchARM64 {
		err = sh.Run("codesign",
			"--remove-signature",
			filepath.Join(m.DistDir, "AutonomousKoi.app", "Contents", "Frameworks", "libcrypto.3.dylib"),
		)
		if err != nil {
			return fmt.Errorf("removing libcrypto signature: %w", err)
		}
		err = sh.Run("codesign",
			"--remove-signature",
			"-s", "D00E79F3D70A4981BC28490E48E91E7430B4A245",
			"--timestamp",
			"-o", "runtime",
			filepath.Join(m.DistDir, "AutonomousKoi.app", "Contents", "Frameworks", "libcrypto.3.dylib"),
		)
		if err != nil {
			return fmt.Errorf("signing libcrypto: %w", err)
		}
	}
	err = sh.Run("codesign",
		"-s", "D00E79F3D70A4981BC28490E48E91E7430B4A245",
		"--entitlements", filepath.Join(m.BaseDir, "ak.entitlements"),
		"--timestamp",
		"-o", "runtime",
		appExecPath,
	)
	if err != nil {
		return fmt.Errorf("signing ak: %w", err)
	}
	return nil
}
