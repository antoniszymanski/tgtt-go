// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	jsonc "github.com/DisposaBoy/JsonConfigReader"
	"github.com/antoniszymanski/sanefmt-go"
	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/internal"
	"github.com/antoniszymanski/tgtt-go/tgtt"
)

type cmdGenerate struct {
	Path string `arg:"" type:"path" default:"tgtt.yml"`
}

func (c *cmdGenerate) Run() error {
	var f *os.File
	var err error
	if c.Path != "-" {
		f, err = os.Open(c.Path)
		if err != nil {
			return err
		}
		defer f.Close() //nolint:errcheck
	} else {
		f = os.Stdin
	}

	var cfg internal.Config
	if err = json.NewDecoder(jsonc.New(f)).Decode(&cfg); err != nil {
		return err
	}

	var formatter tgtt.TsFormatter
	if cfg.Format {
		formatter = func(b []byte) ([]byte, error) {
			return sanefmt.Format(bytes.NewReader(b))
		}
	}

	pkg, err := tgtt.Transpile(tgtt.TranspileOptions{
		PrimaryPackage:    cfg.PrimaryPackage,
		SecondaryPackages: cfg.SecondaryPackages,
		TypeMappings:      cfg.TypeMappings,
		IncludeUnexported: cfg.IncludeUnexported,
	})
	if err != nil {
		return err
	}

	if err = os.MkdirAll(cfg.OutputPath, 0750); err != nil {
		return err
	}
	return pkg.Render(tgtt.PackageRenderOptions{
		Formatter: formatter,
		Write: func(modName string, data []byte) error {
			return os.WriteFile(
				filepath.Join(cfg.OutputPath, modName+".ts"), data, 0600,
			)
		},
	})
}
