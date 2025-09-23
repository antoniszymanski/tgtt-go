// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import "golang.org/x/sync/errgroup"

type TsPackage map[string]*TsModule // Keyed by module name

func (p TsPackage) Index() *TsModule {
	return p["index"]
}

type PackageRenderOptions struct {
	Write func(modName string, data []byte) error
}

func (p TsPackage) Render(opts PackageRenderOptions) error {
	var g errgroup.Group
	for modName, mod := range p {
		g.Go(func() error {
			data, err := mod.Render()
			if err != nil {
				return err
			}
			return opts.Write(modName, data)
		})
	}
	return g.Wait()
}
