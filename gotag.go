package oojson

import (
	"fmt"
	"strings"
)

const (
	JSON_OMITEMPTY = "omitempty"
)

type entry struct {
	Name  string
	Value string
}

type StructTag struct {
	Key             string
	options         []entry
	OptionDelimiter string
	ValueDelimiter  string
}

func newStructTag(key, optDelimiter, valueDelimiter string) *StructTag {
	return &StructTag{
		Key:             key,
		OptionDelimiter: optDelimiter,
		ValueDelimiter:  valueDelimiter,
		options:         []entry{},
	}
}

func (t *StructTag) Set(name, value string) {
	t.options = append(t.options, entry{Name: name, Value: value})
}

func (t *StructTag) Prepend(name, value string) {
	t.options = append([]entry{{Name: name, Value: value}}, t.options...)
}

func (t *StructTag) Unset(name string) {
	newOptions := []entry{}
	for _, opt := range t.options {
		if opt.Name == name {
			continue
		}
		newOptions = append(newOptions, opt)
	}

	t.options = newOptions
}

func (t StructTag) String() string {
	if len(t.options) < 1 {
		return ""
	}
	options := []string{}
	for _, entry := range t.options {
		if len(entry.Value) < 1 {
			options = append(options, entry.Name)
		} else {
			options = append(options, fmt.Sprintf("%v%v%v", entry.Name, t.ValueDelimiter, entry.Value))
		}
	}
	return fmt.Sprintf(`%v:%q`, t.Key, strings.Join(options, t.OptionDelimiter))
}
