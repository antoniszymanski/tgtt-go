/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package main

import (
	"errors"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/goccy/go-yaml"
)

type cli struct {
	Init     cmdInit     `cmd:""`
	Schema   cmdSchema   `cmd:""`
	Generate cmdGenerate `cmd:""`
	Version  cmdVersion  `cmd:""`
}

func formatYAMLError(err error, ignoreEOF bool) error {
	if err == nil || (ignoreEOF && err == io.EOF) {
		return nil
	}
	_, noColor := os.LookupEnv("NO_COLOR")
	return errors.New(yaml.FormatError(err, !noColor, true))
}

func main() {
	var cli cli
	ctx := kong.Parse(&cli,
		kong.Name("tgtt"),
		kong.Description("Transpile Go Types to Typescript"),
		kong.UsageOnError(),
	)
	ctx.FatalIfErrorf(ctx.Run())
}
