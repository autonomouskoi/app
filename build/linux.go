package build

import (
	"path/filepath"

	"github.com/autonomouskoi/mageutil"
	"github.com/magefile/mage/mg"
)

type Linux struct {
	*Base
}

func NewLinux(b *Base) Linux {
	l := Linux{
		Base: b,
	}
	l.ExecPath = filepath.Join(b.DistDir, "autonomouskoi")
	l.GoOS = "linux"
	return l
}

func (l Linux) Release() error {
	mg.Deps(l.Base.compile)
	mg.Deps(l.Base.preRelease)

	zipPath := filepath.Join(l.DistDir, "AutonomousKoi-linux-"+l.GoARCH+"-"+l.ReleaseVersion+".zip")
	return mageutil.ZipFiles(zipPath, map[string]string{
		filepath.Join(l.BaseDir, "LICENSE"): "LICENSE",
		l.ExecPath:                          "autonomouskoi",
	})
}
