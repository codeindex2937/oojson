package oojson

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"go/printer"

	"github.com/fatih/structtag"
	"golang.org/x/exp/maps"
)

var stringIdent = ast.NewIdent("string")
var boolIdent = ast.NewIdent("bool")
var float64Ident = ast.NewIdent("float64")
var jsonNumberIdent = ast.NewIdent("json.Number")
var timeIdent = ast.NewIdent("time.Time")
var anyIdent = ast.NewIdent("any")
var emptyStructIdent = ast.NewIdent("struct{}")
var stringPointerIdent = &ast.StarExpr{X: stringIdent}
var boolPointerIdent = &ast.StarExpr{X: boolIdent}
var float64PointerIdent = &ast.StarExpr{X: float64Ident}
var jsonNumberPointerIdent = &ast.StarExpr{X: jsonNumberIdent}
var timePointerIdent = &ast.StarExpr{X: timeIdent}
var emptyStructPointerIdent = &ast.StarExpr{X: emptyStructIdent}

// goType returns the Go type of v.
func GetGoType(v *Value, options *GoOption) string {
	goType, _ := GetGoAst(v, 0, options)
	buf := bytes.NewBuffer([]byte{})
	printer.Fprint(buf, token.NewFileSet(), goType)
	return buf.String()
}

func GetGoAst(v *Value, observations int, options *GoOption) (ast.Expr, bool) {
	// Determine the number of distinct types observed.
	distinctTypes := 0
	if v.Arrays > 0 {
		distinctTypes++
	}
	if v.Bools > 0 {
		distinctTypes++
	}
	if v.Float64s > 0 {
		distinctTypes++
	}
	if v.Ints > 0 {
		distinctTypes++
	}
	if v.Nulls > 0 {
		distinctTypes++
	}
	if v.Objects > 0 {
		distinctTypes++
	}
	if v.Strings > 0 {
		distinctTypes++
	}

	// Based on the observed distinct types, find the most specific Go type.
	switch {
	case distinctTypes == 1 && v.Arrays > 0:
		fallthrough
	case distinctTypes == 2 && v.Arrays > 0 && v.Nulls > 0:
		elementGoType, _ := GetGoAst(v.ArrayElements, 0, options)
		return &ast.ArrayType{Lbrack: token.NoPos, Elt: elementGoType}, v.Arrays+v.Nulls < observations && v.Emptys == 0
	case distinctTypes == 1 && v.Bools > 0:
		return boolIdent, v.Bools < observations && v.Emptys == 0
	case distinctTypes == 2 && v.Bools > 0 && v.Nulls > 0:
		return boolPointerIdent, false
	case distinctTypes == 1 && v.Float64s > 0:
		return float64Ident, v.Float64s < observations && v.Emptys == 0
	case distinctTypes == 2 && v.Float64s > 0 && v.Nulls > 0:
		return float64PointerIdent, false
	case distinctTypes == 1 && v.Ints > 0:
		return ast.NewIdent(options.intType), v.Ints < observations && v.Emptys == 0
	case distinctTypes == 2 && v.Ints > 0 && v.Nulls > 0:
		return &ast.StarExpr{X: ast.NewIdent(options.intType)}, false
	case distinctTypes == 2 && v.Float64s > 0 && v.Ints > 0:
		omitEmpty := v.Float64s+v.Ints < observations && v.Emptys == 0
		if options.useJSONNumber {
			options.Imports["encoding/json"] = struct{}{}
			return jsonNumberIdent, omitEmpty
		}
		return float64Ident, omitEmpty
	case distinctTypes == 3 && v.Float64s > 0 && v.Ints > 0 && v.Nulls > 0:
		if options.useJSONNumber {
			options.Imports["encoding/json"] = struct{}{}
			return jsonNumberPointerIdent, false
		}
		return float64PointerIdent, false
	case distinctTypes == 1 && v.Objects > 0:
		fallthrough
	case distinctTypes == 2 && v.Objects > 0 && v.Nulls > 0:
		if len(v.ObjectProperties) == 0 {
			switch {
			case observations == 0 && v.Nulls == 0:
				return emptyStructIdent, false
			case v.Nulls > 0:
				return emptyStructPointerIdent, false
			case v.Objects == observations:
				return emptyStructIdent, false
			default:
				return emptyStructPointerIdent, v.Objects < observations
			}
		}
		hasUnparsableProperties := false
		for k := range v.ObjectProperties {
			if strings.ContainsRune(k, ' ') {
				hasUnparsableProperties = true
				break
			}
		}
		if hasUnparsableProperties && !options.skipUnparseableProperties {
			valueGoType, _ := GetGoAst(v.AllObjectProperties, 0, options)
			return &ast.MapType{Map: token.NoPos, Key: stringIdent, Value: valueGoType}, v.Objects+v.Nulls < observations
		}

		structType := &ast.StructType{
			Struct: token.NoPos,
			Fields: &ast.FieldList{
				Opening: token.NoPos,
				Closing: token.NoPos,
				List:    []*ast.Field{},
			},
		}

		properties := maps.Keys(v.ObjectProperties)
		var unparsableProperties []string
		for _, property := range properties {
			if isUnparsableProperty(property) {
				unparsableProperties = append(unparsableProperties, property)
				continue
			}

			goType, observedEmpty := GetGoAst(v.ObjectProperties[property], v.Objects, options)
			var omitEmpty bool
			switch {
			case options.omitEmptyOption == OmitEmptyNever:
				omitEmpty = false
			case options.omitEmptyOption == OmitEmptyAlways:
				omitEmpty = true
			case options.omitEmptyOption == OmitEmptyAuto:
				omitEmpty = observedEmpty
			}

			tags, _ := structtag.Parse("")
			var structTagOptions []string
			if omitEmpty {
				structTagOptions = append(structTagOptions, "omitempty")
			}
			for _, structTagName := range options.structTagNames {
				tag := &structtag.Tag{
					Key:     structTagName,
					Name:    property,
					Options: structTagOptions,
				}
				_ = tags.Set(tag)
			}

			f := &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(property)},
				Type:  goType,
				Tag:   &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`%v`", tags.String())},
			}
			structType.Fields.List = append(structType.Fields.List, f)
		}

		unparsableComments := &ast.CommentGroup{}
		for _, property := range unparsableProperties {
			unparsableComments.List = append(unparsableComments.List, &ast.Comment{
				Slash: token.NoPos,
				Text:  fmt.Sprintf("// %q cannot be unmarshalled into a struct field by encoding/json.", property),
			})
		}
		if len(unparsableProperties) > 0 {
			structType.Fields.List = append(structType.Fields.List, &ast.Field{
				Type:    ast.NewIdent(""),
				Comment: unparsableComments,
			})
		}

		switch {
		case observations == 0:
			return structType, false
		case v.Objects == observations:
			return structType, false
		case v.Objects < observations && v.Nulls == 0:
			return &ast.StarExpr{X: structType}, true
		default:
			return &ast.StarExpr{X: structType}, v.Objects+v.Nulls < observations
		}
	case distinctTypes == 1 && v.Strings > 0 && v.Times == v.Strings:
		options.Imports["time"] = struct{}{}
		return timeIdent, v.Times < observations
	case distinctTypes == 1 && v.Strings > 0:
		return stringIdent, v.Strings < observations && v.Emptys == 0
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0 && v.Times == v.Strings:
		options.Imports["time"] = struct{}{}
		return timePointerIdent, false
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0:
		return stringPointerIdent, false
	default:
		return anyIdent, v.Arrays+v.Bools+v.Float64s+v.Ints+v.Nulls+v.Objects+v.Strings < observations
	}
}

// isUnparsableProperty returns true if key cannot be parsed by encoding/json.
func isUnparsableProperty(key string) bool {
	return strings.ContainsAny(key, ` ",`)
}
