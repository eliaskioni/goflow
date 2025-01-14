package utils

import (
	"reflect"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// IsNil returns whether the given object is nil or an interface to a nil
func IsNil(v interface{}) bool {
	// if v doesn't have a type or value then v == nil
	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)

	// if v is a typed nil pointer then v != nil but the value is nil
	if val.Kind() == reflect.Ptr {
		return val.IsNil()
	}

	return false
}

// Max returns the maximum of two values
func Max[T constraints.Ordered](x, y T) T {
	if x > y {
		return x
	}
	return y
}

// Min returns the minimum of two values
func Min[T constraints.Ordered](x, y T) T {
	if x < y {
		return x
	}
	return y
}

// Set converts a slice to a set (a K > bool map)
func Set[K constraints.Ordered](s []K) map[K]bool {
	m := make(map[K]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

// SortedKeys returns the keys of a set in lexical order
func SortedKeys[K constraints.Ordered](m map[K]bool) []K {
	keys := maps.Keys(m)
	slices.Sort(keys)
	return keys
}

// Typed is an interface of objects that are marshalled as typed envelopes
type Typed interface {
	Type() string
}

// TypedEnvelope can be mixed into envelopes that have a type field
type TypedEnvelope struct {
	Type string `json:"type" validate:"required"`
}

// ReadTypeFromJSON reads a field called `type` from the given JSON
func ReadTypeFromJSON(data []byte) (string, error) {
	t := &TypedEnvelope{}
	if err := UnmarshalAndValidate(data, t); err != nil {
		return "", err
	}
	return t.Type, nil
}
