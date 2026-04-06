//go:build mage
// +build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/autonomouskoi/app/build"
)

func Clean() error {
	base, err := build.NewBase()
	if err != nil {
		return fmt.Errorf("creating base: %w", err)
	}
	return sh.Rm(base.WorkDir)
}

func MacAMD() error {
	base, err := build.NewBase()
	if err != nil {
		return fmt.Errorf("creating base: %w", err)
	}
	base.GoARCH = build.ArchAMD64
	m := build.NewMac(base)
	mg.Deps(m.Release)
	return nil
}

func MacARM() error {
	base, err := build.NewBase()
	if err != nil {
		return fmt.Errorf("creating base: %w", err)
	}
	base.GoARCH = build.ArchARM64
	m := build.NewMac(base)
	mg.Deps(m.Release)
	return nil
}

func LinuxAMD() error {
	base, err := build.NewBase()
	if err != nil {
		return fmt.Errorf("creating base: %w", err)
	}
	base.GoARCH = build.ArchAMD64
	l := build.NewLinux(base)
	mg.Deps(l.Release)
	return nil
}

func LinuxARM() error {
	base, err := build.NewBase()
	if err != nil {
		return fmt.Errorf("creating base: %w", err)
	}
	base.GoARCH = build.ArchARM64
	l := build.NewLinux(base)
	mg.Deps(l.Release)
	return nil
}

func WinAMD() error {
	base, err := build.NewBase()
	if err != nil {
		return fmt.Errorf("creating base: %w", err)
	}
	base.GoARCH = build.ArchAMD64
	w := build.NewWindows(base)
	mg.Deps(w.Release)
	return nil
}
