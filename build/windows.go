package build

import (
	"path/filepath"

	"github.com/autonomouskoi/mageutil"
	"github.com/magefile/mage/mg"
)

type Windows struct {
	*Base
}

func NewWindows(b *Base) Windows {
	w := Windows{
		Base: b,
	}
	w.ExecPath = filepath.Join(b.DistDir, "ak.exe")
	w.GoOS = "windows"
	w.ExtraEnv = map[string]string{
		"CGO_CFLAGS": "-I/mingw64/include",
		"MSYSTEM":    "MINGW64",
		"GOOS":       "windows",
	}
	b.LDFlags = "-H=windowsgui"
	return w
}

func (w Windows) Release() error {
	mg.Deps(w.Base.compile)
	mg.Deps(w.Base.preRelease)

	libCryptoDll := "libcrypto-3-x64.dll"
	libCryptoSrc := filepath.Join(`C:\`, "msys64", "mingw64", "bin", libCryptoDll)
	zipPath := filepath.Join(w.DistDir, "AutonomousKoi-win-"+w.GoARCH+"-"+w.ReleaseVersion+".zip")
	err := mageutil.ZipFiles(zipPath, map[string]string{
		filepath.Join(w.BaseDir, "LICENSE"): "LICENSE",
		"ak.exe":                            w.ExecPath,
		libCryptoSrc:                        libCryptoDll,
	})
	return err
}
