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
