package types

import (
	"math"

	"github.com/nyaruka/goflow/utils"

	"github.com/shopspring/decimal"
)

func init() {
	decimal.MarshalJSONWithoutQuotes = true
}

// XNumber is a whole or fractional number.
//
//   @(1234) -> 1234
//   @(1234.5678) -> 1234.5678
//   @(format_number(1234.5678)) -> 1,234.57
//   @(json(1234.5678)) -> 1234.5678
//
// @type number
type XNumber struct {
	native decimal.Decimal
}

// NewXNumber creates a new XNumber
func NewXNumber(value decimal.Decimal) XNumber {
	return XNumber{native: value}
}

// NewXNumberFromInt creates a new XNumber from the given int
func NewXNumberFromInt(value int) XNumber {
	return NewXNumber(decimal.New(int64(value), 0))
}

// NewXNumberFromInt64 creates a new XNumber from the given int
func NewXNumberFromInt64(value int64) XNumber {
	return NewXNumber(decimal.New(value, 0))
}

// RequireXNumberFromString creates a new XNumber from the given string
func RequireXNumberFromString(value string) XNumber {
	return NewXNumber(decimal.RequireFromString(value))
}

// Describe returns a representation of this type for error messages
func (x XNumber) Describe() string { return x.Render(nil) }

// Truthy determines truthiness for this type
func (x XNumber) Truthy() bool {
	return !x.Equals(XNumberZero)
}

// Render returns the canonical text representation
func (x XNumber) Render(env utils.Environment) string { return x.Native().String() }

// String returns the native string representation of this type
func (x XNumber) String() string { return `XNumber(` + x.Render(nil) + `)` }

// Native returns the native value of this type
func (x XNumber) Native() decimal.Decimal { return x.native }

// Equals determines equality for this type
func (x XNumber) Equals(other XNumber) bool {
	return x.Native().Equals(other.Native())
}

// Compare compares this number to another
func (x XNumber) Compare(other XNumber) int {
	return x.Native().Cmp(other.Native())
}

// MarshalJSON is called when a struct containing this type is marshaled
func (x XNumber) MarshalJSON() ([]byte, error) {
	return x.Native().MarshalJSON()
}

// UnmarshalJSON is called when a struct containing this type is unmarshaled
func (x *XNumber) UnmarshalJSON(data []byte) error {
	nativePtr := &x.native
	return nativePtr.UnmarshalJSON(data)
}

// XNumberZero is the zero number value
var XNumberZero = NewXNumber(decimal.Zero)
var _ XValue = XNumberZero

// ToXNumber converts the given value to a number or returns an error if that isn't possible
func ToXNumber(env utils.Environment, x XValue) (XNumber, XError) {
	if !utils.IsNil(x) {
		switch typed := x.(type) {
		case XError:
			return XNumberZero, typed
		case XNumber:
			return typed, nil
		case XText:
			parsed, err := decimal.NewFromString(typed.Native())
			if err == nil {
				return NewXNumber(parsed), nil
			}
		}
	}

	return XNumberZero, NewXErrorf("unable to convert %s to a number", Describe(x))
}

// ToInteger tries to convert the passed in value to an integer or returns an error if that isn't possible
func ToInteger(env utils.Environment, x XValue) (int, XError) {
	number, err := ToXNumber(env, x)
	if err != nil {
		return 0, err
	}

	intPart := number.Native().IntPart()

	if intPart < math.MinInt32 || intPart > math.MaxInt32 {
		return 0, NewXErrorf("number value %s is out of range for an integer", number.Render(env))
	}

	return int(intPart), nil
}
