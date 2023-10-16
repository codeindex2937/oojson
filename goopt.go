package oojson

import (
	"strings"
	"time"
	"unicode"

	"github.com/fatih/camelcase"
	"golang.org/x/exp/maps"
)

type GoOption struct {
	exportNameFunc            ExportNameFunc
	Imports                   map[string]struct{}
	RegexpValidators          map[string]string
	intType                   string
	omitEmptyOption           OmitEmptyOption
	skipUnparseableProperties bool
	structTagNames            []string
	useJSONNumber             bool
	exportRenames             map[string]string
	timestampFormats          []string
}

var (
	defaultAbbreviations = map[string]bool{
		"API":   true,
		"DB":    true,
		"HTTP":  true,
		"HTTPS": true,
		"ID":    true,
		"JSON":  true,
		"OS":    true,
		"SQL":   true,
		"SSH":   true,
		"URI":   true,
		"URL":   true,
		"XML":   true,
		"YAML":  true,
	}
)

var TimestampFormats = []string{
	time.RFC3339Nano,
	time.DateTime,
	time.Layout,
	time.RFC3339,
	time.DateOnly,
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05.999Z",
	"2006/01/02",
	"2006/01/02 15:04:05",
}

func DefaultGoOption() *GoOption {
	opt := &GoOption{
		Imports: map[string]struct{}{
			"github.com/go-playground/validator/v10": {},
		},
		RegexpValidators:          map[string]string{},
		intType:                   "int",
		omitEmptyOption:           OmitEmptyAuto,
		skipUnparseableProperties: true,
		structTagNames:            []string{"json"},
		useJSONNumber:             false,
	}

	opt.exportNameFunc = func(name string) string {
		if rename, ok := opt.exportRenames[name]; ok {
			return rename
		}
		return DefaultExportNameFunc(name, maps.Clone(defaultAbbreviations))
	}
	return opt
}

// DefaultExportNameFunc returns the exported name for name.
func DefaultExportNameFunc(name string, abbreviations map[string]bool) string {
	components := SplitComponents(name)
	for i, component := range components {
		switch {
		case component == "":
			// do nothing
		case abbreviations[strings.ToUpper(component)]:
			components[i] = strings.ToUpper(component)
		case component == strings.ToUpper(component):
			runes := []rune(component)
			components[i] = string(runes[0]) + strings.ToLower(string(runes[1:]))
		default:
			runes := []rune(component)
			runes[0] = unicode.ToUpper(runes[0])
			components[i] = string(runes)
		}
	}
	runes := []rune(strings.Join(components, ""))
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			runes[i] = '_'
		}
	}
	exportName := string(runes)
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' {
		exportName = "_" + exportName
	}
	return exportName
}

// SplitComponents splits name into components. name may be kebab case, snake
// case, or camel case.
func SplitComponents(name string) []string {
	switch {
	case strings.ContainsRune(name, '-'):
		return strings.Split(name, "-")
	case strings.ContainsRune(name, '_'):
		return strings.Split(name, "_")
	default:
		return camelcase.Split(name)
	}
}
