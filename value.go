package oojson

import (
	"encoding/json"
	"time"
)

// An OmitEmptyOption is an option for handling omitempty.
type OmitEmptyOption int

// omitempty options.
const (
	OmitEmptyNever OmitEmptyOption = iota
	OmitEmptyAlways
	OmitEmptyAuto
)

// An ExportNameFunc returns the exported name for a property.
type ExportNameFunc func(string) string

// An Value describes an observed Value.
type Value struct {
	Observations        int
	Emptys              int
	Arrays              int
	Bools               int
	Float64s            int
	Ints                int
	Nulls               int
	Objects             int
	Strings             int
	Times               int // time.Time is an implicit more specific type than string.
	ArrayElements       *Value
	AllObjectProperties *Value
	ObjectProperties    map[string]*Value
}

// Observe merges a into v.
func (v *Value) Observe(a any) *Value {
	if v == nil {
		v = &Value{}
	}
	v.Observations++
	switch a := a.(type) {
	case []any:
		v.Arrays++
		if len(a) == 0 {
			v.Emptys++
		}
		if v.ArrayElements == nil {
			v.ArrayElements = &Value{}
		}
		for _, e := range a {
			v.ArrayElements = v.ArrayElements.Observe(e)
		}
	case bool:
		v.Bools++
		if !a {
			v.Emptys++
		}
	case float64:
		v.Float64s++
		if a == 0 {
			v.Emptys++
		}
	case int:
		v.Ints++
		if a == 0 {
			v.Emptys++
		}
	case nil:
		v.Nulls++
	case map[string]any:
		v.Objects++
		if len(a) == 0 {
			v.Emptys++
		}
		if v.ObjectProperties == nil {
			v.ObjectProperties = make(map[string]*Value)
		}
		for property, value := range a {
			v.AllObjectProperties = v.AllObjectProperties.Observe(value)
			v.ObjectProperties[property] = v.ObjectProperties[property].Observe(value)
		}
	case string:
		if a == "" {
			v.Emptys++
		}
		if v.Times == v.Strings {
			if _, err := time.Parse(time.RFC3339Nano, a); err == nil {
				v.Times++
			}
		}
		v.Strings++
	case json.Number:
		if _, err := a.Int64(); err == nil {
			v.Ints++
		} else {
			v.Float64s++
		}
	}
	return v
}
