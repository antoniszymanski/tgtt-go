// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package main

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/antoniszymanski/tgtt-go/cmd/tgtt/config"
)

type cmdSchema struct {
	Path string `arg:"" type:"path" default:"tgtt.schema.json"`
}

func (c *cmdSchema) Run() error {
	var f *os.File
	var err error
	if c.Path != "-" {
		if err = os.MkdirAll(filepath.Dir(c.Path), 0750); err != nil {
			return err
		}
		f, err = os.Create(c.Path)
		if err != nil {
			return err
		}
		defer f.Close() //nolint:errcheck
	} else {
		f = os.Stdout
	}

	_, err = f.WriteString(config.Schema())
	return err
}
