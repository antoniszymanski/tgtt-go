// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import "sync"

type TsPackage map[string]*TsModule // Keyed by module name

func (p TsPackage) Index() *TsModule {
	return p["index"]
}

type PackageRenderOptions struct {
	Formatter TsFormatter // through [ModuleRenderOptions]
	Write     func(modName string, data []byte) error
}

func (p TsPackage) Render(opts PackageRenderOptions) error {
	var wg sync.WaitGroup
	var err error
	for modName, mod := range p {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err != nil {
				return
			}
			data, localErr := mod.Render(ModuleRenderOptions{
				Formatter: opts.Formatter,
			})
			if localErr != nil {
				err = localErr
				return
			}
			localErr = opts.Write(modName, data)
			if localErr != nil {
				err = localErr
			}
		}()
	}
	wg.Wait()
	return err
}
