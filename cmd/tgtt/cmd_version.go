/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package main

import (
	"errors"
	"runtime/debug"

	"github.com/alecthomas/kong"
)

type cmdVersion struct{}

func (cmdVersion) Run(ctx *kong.Context) error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("build info not found")
	}

	var revision, time string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if len(setting.Value) >= 8 {
				revision = setting.Value[:8]
			}
		case "vcs.time":
			time = setting.Value
		}
	}
	if revision == "" {
		revision = "unknown"
	}
	if time == "" {
		time = "unknown"
	}

	ctx.Printf(
		`version %s built with %s from %s on %s`,
		info.Main.Version, info.GoVersion, revision, time,
	)
	return nil
}
