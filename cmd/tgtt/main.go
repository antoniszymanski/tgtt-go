/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package main

import (
	"github.com/alecthomas/kong"
)

type cli struct {
	Init     cmdInit     `cmd:""`
	Schema   cmdSchema   `cmd:""`
	Generate cmdGenerate `cmd:""`
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
