package oojson

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/iancoleman/strcase"
	"golang.org/x/exp/maps"
)

const (
	TsBool   = "boolean"
	TsNumber = "number"
	TsAny    = "any"
	TsString = "string"
)

type TsOption struct {
	exportNameFunc ExportNameFunc
	imports        map[string]struct{}
	exportRenames  map[string]string
}

func DefaultTsOption() *TsOption {
	opt := &TsOption{
		imports: make(map[string]struct{}),
	}

	opt.exportNameFunc = func(name string) string {
		if rename, ok := opt.exportRenames[name]; ok {
			return rename
		}
		return strcase.ToLowerCamel(name)
	}
	return opt
}

// goType returns the Go type of v.
func GetTsType(v *Value, name string, indent string, options *TsOption) (string, string) {
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

	subClasses := map[string]string{}

	// Based on the observed distinct types, find the most specific Go type.
	switch {
	case distinctTypes == 1 && v.Arrays > 0:
		fallthrough
	case distinctTypes == 2 && v.Arrays > 0 && v.Nulls > 0:
		elementType, customDefinition := GetTsType(v.ArrayElements, name, indent, options)
		if len(customDefinition) > 0 {
			subClasses[elementType] = customDefinition
		}
		return fmt.Sprintf("%v[]", elementType), customDefinition
	case distinctTypes == 1 && v.Bools > 0:
		return TsBool, ""
	case distinctTypes == 2 && v.Bools > 0 && v.Nulls > 0:
		return TsBool, ""
	case distinctTypes == 1 && v.Float64s > 0:
		return TsNumber, ""
	case distinctTypes == 2 && v.Float64s > 0 && v.Nulls > 0:
		return TsNumber, ""
	case distinctTypes == 1 && v.Ints > 0:
		return TsNumber, ""
	case distinctTypes == 2 && v.Ints > 0 && v.Nulls > 0:
		return TsNumber, ""
	case distinctTypes == 2 && v.Float64s > 0 && v.Ints > 0:
		return TsNumber, ""
	case distinctTypes == 3 && v.Float64s > 0 && v.Ints > 0 && v.Nulls > 0:
		return TsNumber, ""
	case distinctTypes == 1 && v.Objects > 0:
		fallthrough
	case distinctTypes == 2 && v.Objects > 0 && v.Nulls > 0:
		if len(v.ObjectProperties) == 0 {
			return TsAny, ""
		}
		b := &bytes.Buffer{}
		properties := maps.Keys(v.ObjectProperties)
		sort.Strings(properties)
		fmt.Fprintf(b, "type %v = {[key: string]: {\n", name)
		var unparseableProperties []string
		for _, property := range properties {
			if isUnparseableProperty(property) {
				unparseableProperties = append(unparseableProperties, property)
				continue
			}

			subClassType, customCode := GetTsType(v.ObjectProperties[property], strcase.ToCamel(property), indent, options)
			if len(customCode) > 0 {
				subClasses[subClassType] = customCode
			}
			fmt.Fprintf(b, "%v%s: %s;\n", indent, strcase.ToLowerCamel(property), subClassType)
		}
		fmt.Fprintf(b, "}}\n\n")

		for _, property := range unparseableProperties {
			fmt.Fprintf(b, "// %q cannot be unmarshalled into a struct field by encoding/json.\n", property)
		}

		for _, code := range subClasses {
			fmt.Fprintf(b, "%s\n", code)
		}

		return name, b.String()
	case distinctTypes == 1 && v.Strings > 0 && v.Times == v.Strings:
		return TsString, ""
	case distinctTypes == 1 && v.Strings > 0:
		return TsString, ""
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0 && v.Times == v.Strings:
		return TsString, ""
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0:
		return TsString, ""
	default:
		return TsAny, ""
	}
}
