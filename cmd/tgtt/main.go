// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package main

import "github.com/alecthomas/kong"

var cli struct {
	Init     cmdInit     `cmd:""`
	Schema   cmdSchema   `cmd:""`
	Generate cmdGenerate `cmd:""`
	Version  cmdVersion  `cmd:""`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.Name("tgtt"),
		kong.Description("Transpile Go Types to Typescript"),
		kong.UsageOnError(),
	)
	ctx.FatalIfErrorf(ctx.Run())
}
