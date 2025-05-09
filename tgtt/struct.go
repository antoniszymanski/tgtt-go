/*
This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at https://mozilla.org/MPL/2.0/.
*/

package tgtt

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
)

func (t *transpiler) transpileStruct(typ topLevel, mod *Module) string {
	path := t.getPkgPath(typ.Obj())
	typeMapping, ok := t.TypeMappings[path]
	if ok {
		return fmt.Sprintf(
			`export type %s%s = %s`,
			typ.Obj().Name(),
			t.transpileTypeParams(typ.TypeParams(), mod),
			typeMapping,
		)
	}

	s := parseStruct(typ.Underlying().(*types.Struct))
	return fmt.Sprintf(
		`export interface %s%s%s %s`,
		typ.Obj().Name(),
		t.transpileTypeParams(typ.TypeParams(), mod),
		t.transpileExtends(s, mod),
		t.transpileStructBody(s, mod),
	)
}

func (t *transpiler) transpileExtends(s structData, mod *Module) string {
	var extends []string
	for _, field := range s.Embedded {
		if _, ok := field.Type.Underlying().(*types.Struct); !ok {
			continue
		}
		typStr := t.transpileType(field.Type, mod)
		typStr, found := strings.CutSuffix(typStr, " | null")
		if found {
			typStr = fmt.Sprintf(`Partial<%s>`, typStr)
		}
		extends = append(extends, typStr)
	}

	if len(extends) == 0 {
		return ""
	}
	return " " + "extends " + strings.Join(extends, ", ")
}

func (t *transpiler) transpileStructBody(s structData, mod *Module) string {
	var sb strings.Builder

	sb.WriteByte('{')
	if len(s.Fields) > 0 {
		sb.WriteByte(' ')
	}
	for _, field := range s.Fields {
		format := `%s: %s; `
		if field.Optional {
			format = `%s?: %s; `
		}
		fmt.Fprintf(
			&sb,
			format,
			strconv.Quote(field.Name),
			t.transpileType(field.Type, mod),
		)
	}
	sb.WriteByte('}')

	return sb.String()
}

type structData struct {
	Embedded, Fields []fieldData
}

type fieldData struct {
	Name     string
	Optional bool
	Type     types.Type
}

func parseStruct(typ *types.Struct) structData {
	var s structData
	for i := range typ.NumFields() {
		field := typ.Field(i)
		if !field.Exported() {
			continue
		}

		var f fieldData
		func() {
			tags, err := structtag.Parse(typ.Tag(i))
			if err != nil {
				f.Name = field.Name()
				return
			}

			tag, err := tags.Get("json")
			if err != nil {
				f.Name = field.Name()
				return
			}

			f.Name = tag.Name
			f.Optional = tag.HasOption("omitempty")
		}()
		if f.Name == "-" {
			continue
		}

		f.Type = field.Type()
		if !field.Embedded() {
			s.Fields = append(s.Fields, f)
		} else {
			s.Embedded = append(s.Embedded, f)
		}
	}

	return s
}
