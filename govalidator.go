package oojson

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
)

// goType returns the Go type of v.
func GetGoValidator(v *Value, observations int, options *GoOption) (string, map[string]*StructTag) {
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

	tagMap := make(map[string]*StructTag)
	validatorTag := newStructTag("validate", ",", "=")
	jsonTag := newStructTag("json", ",", "=")
	tagMap[jsonTag.Key] = jsonTag
	tagMap[validatorTag.Key] = validatorTag

	// Based on the observed distinct types, find the most specific Go type.
	switch {
	case distinctTypes == 1 && v.Arrays > 0:
		fallthrough
	case distinctTypes == 2 && v.Arrays > 0 && v.Nulls > 0:
		elementGoType, elementTagMap := GetGoValidator(v.ArrayElements, 0, options)
		if v.Arrays+v.Nulls < observations && v.Emptys == 0 {
			jsonTag.Set(JSON_OMITEMPTY, "")
		}
		elementTagMap["validate"].Prepend("dive", "")
		elementTagMap["validate"].Prepend("required", "")
		return "[]" + elementGoType, elementTagMap
	case distinctTypes == 1 && v.Bools > 0:
		tagMap["validate"].Set("required", "")
		if v.Bools < observations && v.Emptys == 0 {
			jsonTag.Set(JSON_OMITEMPTY, "")
		}
		return "bool", tagMap
	case distinctTypes == 2 && v.Bools > 0 && v.Nulls > 0:
		return "*bool", tagMap
	case distinctTypes == 1 && v.Float64s > 0:
		tagMap["validate"].Set("required", "")
		if v.Float64s < observations && v.Emptys == 0 {
			jsonTag.Set(JSON_OMITEMPTY, "")
		}
		return "float64", tagMap
	case distinctTypes == 2 && v.Float64s > 0 && v.Nulls > 0:
		return "*float64", tagMap
	case distinctTypes == 1 && v.Ints > 0:
		tagMap["validate"].Set("required", "")
		if v.Ints < observations && v.Emptys == 0 {
			jsonTag.Set(JSON_OMITEMPTY, "")
		}
		return options.intType, tagMap
	case distinctTypes == 2 && v.Ints > 0 && v.Nulls > 0:
		return "*" + options.intType, tagMap
	case distinctTypes == 2 && v.Float64s > 0 && v.Ints > 0:
		omitEmpty := v.Float64s+v.Ints < observations && v.Emptys == 0
		if omitEmpty {
			jsonTag.Set(JSON_OMITEMPTY, "")
		}
		if options.useJSONNumber {
			options.Imports["encoding/json"] = struct{}{}
			return "json.Number", tagMap
		}
		tagMap["validate"].Set("required", "")
		return "float64", tagMap
	case distinctTypes == 3 && v.Float64s > 0 && v.Ints > 0 && v.Nulls > 0:
		if options.useJSONNumber {
			options.Imports["encoding/json"] = struct{}{}
			return "*json.Number", tagMap
		}
		return "*float64", tagMap
	case distinctTypes == 1 && v.Objects > 0:
		fallthrough
	case distinctTypes == 2 && v.Objects > 0 && v.Nulls > 0:
		if len(v.ObjectProperties) == 0 {
			switch {
			case observations == 0 && v.Nulls == 0:
				tagMap["validate"].Set("required", "")
				return "struct{}", tagMap
			case v.Nulls > 0:
				return "*struct{}", tagMap
			case v.Objects == observations:
				tagMap["validate"].Set("required", "")
				return "struct{}", tagMap
			default:
				tagMap["validate"].Set("required", "")
				if v.Objects < observations {
					jsonTag.Set(JSON_OMITEMPTY, "")
				}
				return "*struct{}", tagMap
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
			valueGoType, validatorMap := GetGoValidator(v.AllObjectProperties, 0, options)
			if v.Objects+v.Nulls < observations {
				jsonTag.Set(JSON_OMITEMPTY, "")
			}
			validatorMap["validate"].Prepend("dive", "")
			tagMap["validate"].Prepend("required", "")
			return "map[string]" + valueGoType, validatorMap
		}
		b := &bytes.Buffer{}
		properties := maps.Keys(v.ObjectProperties)
		sort.Strings(properties)
		fmt.Fprintf(b, "struct {\n")
		var unparseableProperties []string
		for _, property := range properties {
			if isUnparsableProperty(property) {
				unparseableProperties = append(unparseableProperties, property)
				continue
			}
			goType, tagMap := GetGoValidator(v.ObjectProperties[property], v.Objects, options)
			tagMap["json"].Set(property, "")

			var omitEmpty bool
			switch {
			case options.omitEmptyOption == OmitEmptyNever:
				tagMap["json"].Unset(JSON_OMITEMPTY)
			case options.omitEmptyOption == OmitEmptyAlways:
				tagMap["json"].Set(JSON_OMITEMPTY, "")
			case options.omitEmptyOption == OmitEmptyAuto:
				// use return value
			}

			if omitEmpty {
				jsonTag.Set(JSON_OMITEMPTY, "")
			}

			fmt.Fprintf(b, "%s %s `%s`\n", options.exportNameFunc(property), goType, getTagsString(tagMap))
		}

		for _, property := range unparseableProperties {
			fmt.Fprintf(b, "// %q cannot be unmarshalled into a struct field by encoding/json.\n", property)
		}
		fmt.Fprintf(b, "}")
		switch {
		case observations == 0:
			tagMap["validate"].Set("required", "")
			return b.String(), tagMap
		case v.Objects == observations:
			tagMap["validate"].Set("required", "")
			return b.String(), tagMap
		case v.Objects < observations && v.Nulls == 0:
			jsonTag.Set(JSON_OMITEMPTY, "")
			return "*" + b.String(), tagMap
		default:
			if v.Objects+v.Nulls < observations {
				jsonTag.Set(JSON_OMITEMPTY, "")
			}
			return "*" + b.String(), tagMap
		}
	case distinctTypes == 1 && v.Strings > 0 && v.Times == v.Strings:
		safeTagName := getSafeTagName(v.TimestampFormat)
		options.RegexpValidators[safeTagName] = regexp.MustCompile(`\d`).ReplaceAllString(v.TimestampFormat, `\d`)
		validatorTag.Set(safeTagName, "")
		if v.Times < observations {
			jsonTag.Set(JSON_OMITEMPTY, "")
		}
		return "string", tagMap
	case distinctTypes == 1 && v.Strings > 0:
		if v.Strings < observations && v.Emptys == 0 {
			jsonTag.Set(JSON_OMITEMPTY, "")
		}
		return "string", tagMap
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0 && v.Times == v.Strings:
		safeTagName := getSafeTagName(v.TimestampFormat)
		options.RegexpValidators[safeTagName] = regexp.MustCompile(`\d`).ReplaceAllString(v.TimestampFormat, `\d`)
		validatorTag.Set(safeTagName, "")
		tagMap["validate"].Set("required", "")
		return "*string", tagMap
	case distinctTypes == 2 && v.Strings > 0 && v.Nulls > 0:
		return "*string", tagMap
	default:
		if v.Arrays+v.Bools+v.Float64s+v.Ints+v.Nulls+v.Objects+v.Strings < observations {
			jsonTag.Set(JSON_OMITEMPTY, "")
		}
		return "any", tagMap
	}
}

func getTagsString(tags map[string]*StructTag) string {
	tagsString := []string{}
	for _, tag := range tags {
		s := tag.String()
		if len(s) < 1 {
			continue
		}
		tagsString = append(tagsString, s)
	}

	return strings.Join(tagsString, " ")
}

func getSafeTagName(s string) string {
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "_")
	return regexp.MustCompile(`[^\w]+`).ReplaceAllString(s, "")
}
