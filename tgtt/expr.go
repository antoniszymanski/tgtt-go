// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strconv"
)

func TranspileExpr(x string) (string, error) {
	expr, err := parser.ParseExpr(x)
	if err != nil {
		return "", err
	}
	b, err := transpileExpr(nil, expr)
	return bytesToString(b), err
}

func transpileExpr(dst []byte, expr ast.Expr) ([]byte, error) {
	switch expr := expr.(type) {
	case *ast.ArrayType:
		return transpileArrayType(dst, expr)
	case *ast.BadExpr:
		return transpileBadExpr(dst, expr)
	// case *ast.BasicLit:
	// case *ast.BinaryExpr:
	// case *ast.CallExpr:
	// case *ast.ChanType:
	// case *ast.CompositeLit:
	// case *ast.Ellipsis:
	// case *ast.FuncLit:
	// case *ast.FuncType:
	case *ast.Ident:
		return transpileIdent(dst, expr)
	// case *ast.IndexExpr:
	// case *ast.IndexListExpr:
	case *ast.InterfaceType:
		return transpileInterfaceType(dst, expr)
	// case *ast.KeyValueExpr:
	case *ast.MapType:
		return transpileMapType(dst, expr)
	case *ast.ParenExpr:
		return transpileParenExpr(dst, expr)
	case *ast.SelectorExpr:
		return transpileSelectorExpr(dst, expr)
	// case *ast.SliceExpr:
	case *ast.StarExpr:
		return transpileStarExpr(dst, expr)
	case *ast.StructType:
		return transpileStructType(dst, expr)
	// case *ast.TypeAssertExpr:
	// case *ast.UnaryExpr:
	default:
		err := reflect.TypeOf(expr).Elem().Name() + ": unsupported expression type"
		return nil, errors.New(err)
	}
}

func transpileArrayType(dst []byte, expr *ast.ArrayType) ([]byte, error) {
	dst, err := transpileExpr(dst, expr.Elt)
	if err != nil {
		return nil, err
	}
	dst = append(dst, "[]"...)
	return dst, nil
}

func transpileBadExpr(_ []byte, expr *ast.BadExpr) ([]byte, error) {
	return nil, &ErrBadExpr{From: expr.From, To: expr.To}
}

type ErrBadExpr struct {
	From, To token.Pos
}

func (e *ErrBadExpr) Error() string {
	return fmt.Sprintf("BadExpr: syntax error found at position %d to %d", e.From, e.To)
}

func transpileIdent(dst []byte, expr *ast.Ident) ([]byte, error) {
	return append(dst, expr.Name...), nil
}

func transpileInterfaceType(dst []byte, _ *ast.InterfaceType) ([]byte, error) {
	return append(dst, "any"...), nil
}

func transpileMapType(dst []byte, expr *ast.MapType) ([]byte, error) {
	dst = append(dst, "{ [key in string]: "...)
	dst, err := transpileExpr(dst, expr.Value)
	if err != nil {
		return nil, err
	}
	dst = append(dst, " }"...)
	return dst, nil
}

func transpileParenExpr(dst []byte, expr *ast.ParenExpr) ([]byte, error) {
	return transpileExpr(dst, expr.X)
}

func transpileSelectorExpr(dst []byte, expr *ast.SelectorExpr) ([]byte, error) {
	dst, err := transpileExpr(dst, expr.X)
	if err != nil {
		return nil, err
	}
	dst = append(dst, '.')
	dst = append(dst, expr.Sel.Name...)
	return dst, nil
}

func transpileStarExpr(dst []byte, expr *ast.StarExpr) ([]byte, error) {
	dst, err := transpileExpr(dst, expr.X)
	if err != nil {
		return nil, err
	}
	if !bytes.HasSuffix(dst, []byte(" | null")) {
		dst = append(dst, " | null"...)
	}
	return dst, nil
}

func transpileStructType(dst []byte, expr *ast.StructType) (_ []byte, err error) {
	s := parseStructType(expr)
	dst = append(dst, '{')
	if len(s.Fields) > 0 {
		dst = append(dst, ' ')
	}
	for i, field := range s.Fields {
		dst = strconv.AppendQuote(dst, field.Name)
		if field.Optional {
			dst = append(dst, '?')
		}
		dst = append(dst, ": "...)
		dst, err = transpileExpr(dst, field.Type)
		if err != nil {
			return nil, err
		}
		if i < len(s.Fields)-1 {
			dst = append(dst, ';')
		}
		dst = append(dst, ' ')
	}
	dst = append(dst, '}')
	for _, embedded := range s.Embedded {
		dst = append(dst, " & "...)
		i := len(dst)
		dst, err = transpileExpr(dst, embedded)
		if err != nil {
			return nil, err
		}
		var found bool
		dst, found = bytes.CutSuffix(dst, []byte(" | null"))
		if found {
			dst = slices.Insert(dst, i, []byte("Partial<")...)
			dst = append(dst, '>')
		}
	}
	return dst, nil
}

func parseStructType(expr *ast.StructType) structInfo[ast.Expr] {
	var s structInfo[ast.Expr]
	for _, field := range expr.Fields.List {
		var names []string
		if len(field.Names) == 0 {
			names = []string{""}
		} else {
			names = make([]string, 0, len(field.Names))
			for _, name := range field.Names {
				if name.IsExported() {
					names = append(names, name.Name)
				}
			}
		}
		if len(names) == 0 {
			continue
		}
		var tag string
		if field.Tag != nil {
			tag, _ = strconv.Unquote(field.Tag.Value)
		}
		parse := parseFieldTag[ast.Expr](tag)
		for _, name := range names {
			f := fieldInfo[ast.Expr]{Name: name, Type: field.Type}
			if parse(&f) {
				continue
			}
			if f.Name == "" {
				s.Embedded = append(s.Embedded, f.Type)
			} else {
				s.Fields = append(s.Fields, f)
			}
		}
	}
	return s
}
