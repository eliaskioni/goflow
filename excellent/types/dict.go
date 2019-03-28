package types

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nyaruka/goflow/utils"
)

// XDict is a map primitive in Excellent expressions
type XDict interface {
	XPrimitive
	XResolvable
	XLengthable

	Get(string) XValue
	Put(string, XValue)
	Keys() []string
}

type xdict struct {
	values map[string]XValue
}

// NewXDict returns a new map with the given items
func NewXDict(values map[string]XValue) XDict {
	return &xdict{
		values: values,
	}
}

// NewEmptyXDict returns a new empty map
func NewEmptyXDict() XDict {
	return &xdict{
		values: make(map[string]XValue),
	}
}

// Describe returns a representation of this type for error messages
func (x *xdict) Describe() string { return "dict" }

// Reduce returns the primitive version of this type (i.e. itself)
func (x *xdict) Reduce(env utils.Environment) XPrimitive { return x }

// ToXText converts this type to text
func (x *xdict) ToXText(env utils.Environment) XText {
	// get our keys sorted A-Z
	sortedKeys := x.Keys()
	sort.Strings(sortedKeys)

	pairs := make([]string, 0, x.Length())
	for _, k := range sortedKeys {
		vAsText, xerr := ToXText(env, x.values[k])
		if xerr != nil {
			vAsText = xerr.ToXText(env)
		}

		pairs = append(pairs, fmt.Sprintf("%s: %s", k, vAsText))
	}
	return NewXText("{" + strings.Join(pairs, ", ") + "}")
}

// ToXBoolean converts this type to a bool
func (x *xdict) ToXBoolean(env utils.Environment) XBoolean {
	return NewXBoolean(len(x.values) > 0)
}

// ToXJSON is called when this type is passed to @(json(...))
func (x *xdict) ToXJSON(env utils.Environment) XText {
	marshaled := make(map[string]json.RawMessage, len(x.values))
	for k, v := range x.values {
		asJSON, err := ToXJSON(env, v)
		if err == nil {
			marshaled[k] = json.RawMessage(asJSON.Native())
		}
	}
	return MustMarshalToXText(marshaled)
}

// MarshalJSON converts this type to internal JSON
func (x *xdict) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.values)
}

// Length is called when the length of this object is requested in an expression
func (x *xdict) Length() int {
	return len(x.values)
}

func (x *xdict) Resolve(env utils.Environment, key string) XValue {
	val, found := x.values[key]
	if !found {
		return NewXResolveError(x, key)
	}
	return val
}

// Get retrieves the named item from this dict
func (x *xdict) Get(key string) XValue {
	return x.values[key]
}

// Put adds the given item to this dict
func (x *xdict) Put(key string, value XValue) {
	x.values[key] = value
}

// Keys returns the keys of this dict
func (x *xdict) Keys() []string {
	keys := make([]string, 0, len(x.values))
	for key := range x.values {
		keys = append(keys, key)
	}
	return keys
}

// String returns the native string representation of this type
func (x *xdict) String() string { return x.ToXText(nil).Native() }

var _ XDict = (*xdict)(nil)
var _ json.Marshaler = (*xdict)(nil)