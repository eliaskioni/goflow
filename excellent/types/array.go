package types

import (
	"encoding/json"
)

// XArray is an array primitive in Excellent expressions
type XArray interface {
	XPrimitive
	XIndexable

	Append(XValue)
}

type xarray struct {
	values []XValue
}

// NewXArray returns a new array with the given items
func NewXArray(values ...XValue) XArray {
	return &xarray{values: values}
}

// Reduce returns the primitive version of this type (i.e. itself)
func (a *xarray) Reduce() XPrimitive { return a }

// ToString converts this type to a string
func (a *xarray) ToString() XString {
	strs := make([]XString, len(a.values))
	for i := range a.values {
		strs[i] = a.values[i].Reduce().ToString()
	}
	return MustMarshalToXString(strs)
}

// ToBool converts this type to a bool
func (a *xarray) ToBool() XBool {
	return len(a.values) > 0
}

// ToJSON converts this type to JSON
func (a *xarray) ToJSON() XString {
	marshaled := make([]json.RawMessage, len(a.values))
	for i := range a.values {
		marshaled[i] = json.RawMessage(a.values[i].ToJSON())
	}
	return MustMarshalToXString(marshaled)
}

// Index is called when this object is indexed into in an expression
func (a *xarray) Index(index int) XValue {
	return a.values[index]
}

// Length is called when the length of this object is requested in an expression
func (a *xarray) Length() int {
	return len(a.values)
}

// Append adds the given item to this array
func (a *xarray) Append(value XValue) {
	a.values = append(a.values, value)
}

var _ XArray = (*xarray)(nil)