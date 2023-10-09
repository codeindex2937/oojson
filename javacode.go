package oojson

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/iancoleman/strcase"
	"golang.org/x/exp/maps"
)

const (
	JavaBool   = "Boolean"
	JavaFloat  = "Float"
	JavaInt    = "Integer"
	JavaAny    = "Object"
	JavaString = "String"
	JavaTime   = "Date"
)

type JavaOption struct {
	exportNameFunc ExportNameFunc
	imports        map[string]struct{}
	exportRenames  map[string]string
}

func DefaultJavaOption() *JavaOption {
	opt := &JavaOption{
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
func GetJavaType(v *Value, name string, indent string, options *JavaOption) (string, string) {
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
		elementJavaType, customDefinition := GetJavaType(v.ArrayElements, name, indent, options)
		options.imports["java.util.List"] = struct{}{}
		if len(customDefinition) > 0 {
			subClasses[elementJavaType] = customDefinition
		}
		return fmt.Sprintf("List<%v>", elementJavaType), customDefinition
	case distinctTypes == 1 && v.Bools > 0:
		return JavaBool, ""
	case distinctTypes == 2 && v.Bools > 0 && v.Nulls > 0:
		return JavaBool, ""
	case distinctTypes == 1 && v.Float64s > 0:
		return JavaFloat, ""
	case distinctTypes == 2 && v.Float64s > 0 && v.Nulls > 0:
		return JavaFloat, ""
	case distinctTypes == 1 && v.Ints > 0:
		return JavaInt, ""
	case distinctTypes == 2 && v.Ints > 0 && v.Nulls > 0:
		return JavaInt, ""
	case distinctTypes == 2 && v.Float64s > 0 && v.Ints > 0:
		return JavaFloat, ""
	case distinctTypes == 3 && v.Float64s > 0 && v.Ints > 0 && v.Nulls > 0:
		return JavaFloat, ""
	case distinctTypes == 1 && v.Objects > 0:
		fallthrough
	case distinctTypes == 2 && v.Objects > 0 && v.Nulls > 0:
		if len(v.ObjectProperties) == 0 {
			return JavaAny, ""
		}
		b := &bytes.Buffer{}
		properties := maps.Keys(v.ObjectProperties)
		sort.Strings(properties)
		fmt.Fprintf(b, "class %v {\n", name)
		var unparseableProperties []string
		for _, property := range properties {
			if isUnparseableProperty(property) {
				unparseableProperties = append(unparseableProperties, property)
				continue
			}

			subClassType, customCode := GetJavaType(v.ObjectProperties[property], strcase.ToCamel(property), indent, options)
			if len(customCode) > 0 {
				subClasses[subClassType] = customCode
			}
			fmt.Fprintf(b, "%vprivate %s %s;\n", indent, subClassType, strcase.ToLowerCamel(property))
		}
		for _, property := range unparseableProperties {
			fmt.Fprintf(b, "// %q cannot be unmarshalled into a struct field by encoding/json.\n", property)
		}

		for _, code := range subClasses {
			fmt.Fprintf(b, "%v@Data\n%vpublic static %s\n", indent, indent, strings.ReplaceAll(code, "\n", "\n"+indent))
		}
		fmt.Fprintf(b, "}")

		return name, b.String()
	case distinctTypes == 1 && v.Strings > 0 && v.Times == v.Strings:
		options.imports["java.util.Date"] = struct{}{}
		return JavaTime, ""
	case distinctTypes == 1 && v.Strings > 0:
		return JavaString, ""
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0 && v.Times == v.Strings:
		options.imports["java.util.Date"] = struct{}{}
		return JavaTime, ""
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0:
		return JavaString, ""
	default:
		return JavaAny, ""
	}
}
