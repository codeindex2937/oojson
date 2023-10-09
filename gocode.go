package oojson

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/structtag"
	"golang.org/x/exp/maps"
)

// goType returns the Go type of v.
func GetGoType(v *Value, observations int, options *GoOption) (string, bool) {
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
		elementGoType, _ := GetGoType(v.ArrayElements, 0, options)
		return "[]" + elementGoType, v.Arrays+v.Nulls < observations && v.Emptys == 0
	case distinctTypes == 1 && v.Bools > 0:
		return "bool", v.Bools < observations && v.Emptys == 0
	case distinctTypes == 2 && v.Bools > 0 && v.Nulls > 0:
		return "*bool", false
	case distinctTypes == 1 && v.Float64s > 0:
		return "float64", v.Float64s < observations && v.Emptys == 0
	case distinctTypes == 2 && v.Float64s > 0 && v.Nulls > 0:
		return "*float64", false
	case distinctTypes == 1 && v.Ints > 0:
		return options.intType, v.Ints < observations && v.Emptys == 0
	case distinctTypes == 2 && v.Ints > 0 && v.Nulls > 0:
		return "*" + options.intType, false
	case distinctTypes == 2 && v.Float64s > 0 && v.Ints > 0:
		omitEmpty := v.Float64s+v.Ints < observations && v.Emptys == 0
		if options.useJSONNumber {
			options.imports["encoding/json"] = struct{}{}
			return "json.Number", omitEmpty
		}
		return "float64", omitEmpty
	case distinctTypes == 3 && v.Float64s > 0 && v.Ints > 0 && v.Nulls > 0:
		if options.useJSONNumber {
			options.imports["encoding/json"] = struct{}{}
			return "*json.Number", false
		}
		return "*float64", false
	case distinctTypes == 1 && v.Objects > 0:
		fallthrough
	case distinctTypes == 2 && v.Objects > 0 && v.Nulls > 0:
		if len(v.ObjectProperties) == 0 {
			switch {
			case observations == 0 && v.Nulls == 0:
				return "struct{}", false
			case v.Nulls > 0:
				return "*struct{}", false
			case v.Objects == observations:
				return "struct{}", false
			default:
				return "*struct{}", v.Objects < observations
			}
		}
		hasUnparseableProperties := false
		for k := range v.ObjectProperties {
			if strings.ContainsRune(k, ' ') {
				hasUnparseableProperties = true
				break
			}
		}
		if hasUnparseableProperties && !options.skipUnparseableProperties {
			valueGoType, _ := GetGoType(v.AllObjectProperties, 0, options)
			return "map[string]" + valueGoType, v.Objects+v.Nulls < observations
		}
		b := &bytes.Buffer{}
		properties := maps.Keys(v.ObjectProperties)
		sort.Strings(properties)
		fmt.Fprintf(b, "struct {\n")
		var unparseableProperties []string
		for _, property := range properties {
			if isUnparseableProperty(property) {
				unparseableProperties = append(unparseableProperties, property)
				continue
			}
			goType, observedEmpty := GetGoType(v.ObjectProperties[property], v.Objects, options)
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

			fmt.Fprintf(b, "%s %s `%s`\n", options.exportNameFunc(property), goType, tags)
		}
		for _, property := range unparseableProperties {
			fmt.Fprintf(b, "// %q cannot be unmarshalled into a struct field by encoding/json.\n", property)
		}
		fmt.Fprintf(b, "}")
		switch {
		case observations == 0:
			return b.String(), false
		case v.Objects == observations:
			return b.String(), false
		case v.Objects < observations && v.Nulls == 0:
			return "*" + b.String(), true
		default:
			return "*" + b.String(), v.Objects+v.Nulls < observations
		}
	case distinctTypes == 1 && v.Strings > 0 && v.Times == v.Strings:
		options.imports["time"] = struct{}{}
		return "time.Time", v.Times < observations
	case distinctTypes == 1 && v.Strings > 0:
		return "string", v.Strings < observations && v.Emptys == 0
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0 && v.Times == v.Strings:
		options.imports["time"] = struct{}{}
		return "*time.Time", false
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0:
		return "*string", false
	default:
		return "any", v.Arrays+v.Bools+v.Float64s+v.Ints+v.Nulls+v.Objects+v.Strings < observations
	}
}

// isUnparseableProperty returns true if key cannot be parsed by encoding/json.
func isUnparseableProperty(key string) bool {
	return strings.ContainsAny(key, ` ",`)
}
