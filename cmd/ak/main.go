package main

import (
	"github.com/autonomouskoi/akcore/exe"
	_ "github.com/autonomouskoi/banter"
	_ "github.com/autonomouskoi/trackstar"
	_ "github.com/autonomouskoi/trackstar/overlay"
	_ "github.com/autonomouskoi/trackstar/stagelinq"
	_ "github.com/autonomouskoi/trackstar/twitchchat"
	_ "github.com/autonomouskoi/twitch"
)

func main() {
	exe.Main()
}
