package app

import (
	_ "embed"
	"strings"

	"github.com/autonomouskoi/akcore"
)

//go:embed VERSION
var Version string

func init() {
	akcore.Version = strings.TrimSpace(Version)
}
