package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Resolvable is the interface for objects in the context which can be keyed into, e.g. foo.bar
type Resolvable interface {
	Resolve(key string) interface{}
}

// Lengthable is the interface for objects in the context which have a length
type Lengthable interface {
	Length() int
}

// Indexable is the interface for objects in the context which can be indexed into, e.g. foo.0. Such objects
// also need to be lengthable so that the engine knows what is a valid index and what isn't.
type Indexable interface {
	Lengthable

	Index(index int) interface{}
}

// Atomizable is the interface for objects in the context which can reduce themselves to an XAtom primitive
type Atomizable interface {
	Atomize() interface{}
}

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

// ResolveVariable will resolve the passed in string variable given in dot notation and return
// the value as defined by the VariableResolver passed in.
//
// Example syntaxes:
//      foo.bar.0  - 0th element of bar slice within foo, could also be "0" key in bar map within foo
//      foo.bar[0] - same as above
func ResolveVariable(env Environment, variable interface{}, key string) interface{} {
	var err error

	err, isErr := variable.(error)
	if isErr {
		return err
	}

	// self referencing
	if key == "" {
		return variable
	}

	// strip leading '.'
	if key[0] == '.' {
		key = key[1:]
	}

	rest := key
	for rest != "" {
		key, rest = popNextVariable(rest)

		if IsNil(variable) {
			return fmt.Errorf("can't resolve key '%s' of nil", key)
		}

		// is our key numeric?
		index, err := strconv.Atoi(key)
		if err == nil {
			indexable, isIndexable := variable.(Indexable)
			if isIndexable {
				if index >= indexable.Length() || index < -indexable.Length() {
					return fmt.Errorf("index %d out of range for %d items", index, indexable.Length())
				}
				if index < 0 {
					index += indexable.Length()
				}
				variable = indexable.Index(index)
				continue
			}
		}

		resolver, isResolver := variable.(Resolvable)

		// look it up in our resolver
		if isResolver {
			variable = resolver.Resolve(key)

			err, isErr := variable.(error)
			if isErr {
				return err
			}

		} else {
			return fmt.Errorf("can't resolve key '%s' of type %s", key, reflect.TypeOf(variable))
		}
	}

	// check what we are returning is a type that expressions understand
	_, _, err = ToXAtom(env, variable)
	if err != nil {
		_, isAtomizable := variable.(Atomizable)
		if !isAtomizable {
			panic(fmt.Sprintf("key '%s' of resolved to usupported type %s", key, reflect.TypeOf(variable)))
		}
	}

	return variable
}

// popNextVariable pops the next variable off our string:
//     foo.bar.baz -> "foo", "bar.baz"
//     foo[0].bar -> "foo", "[0].baz"
//     foo.0.bar -> "foo", "0.baz"
//     [0].bar -> "0", "bar"
//     foo["my key"] -> "foo", "my key"
func popNextVariable(input string) (string, string) {
	var keyStart = 0
	var keyEnd = -1
	var restStart = -1

	for i, c := range input {
		if i == 0 && c == '[' {
			keyStart++
		} else if c == '[' {
			keyEnd = i
			restStart = i
			break
		} else if c == ']' {
			keyEnd = i
			restStart = i + 1
			break
		} else if c == '.' {
			keyEnd = i
			restStart = i + 1
			break
		}
	}

	if keyEnd == -1 {
		return input, ""
	}

	key := strings.Trim(input[keyStart:keyEnd], "\"")
	rest := input[restStart:]

	return key, rest
}

type mapResolver struct {
	values map[string]interface{}
}

// NewMapResolver returns a simple resolver that resolves variables according to the values
// passed in
func NewMapResolver(values map[string]interface{}) Resolvable {
	return &mapResolver{
		values: values,
	}
}

// Resolve resolves the given key when this map is referenced in an expression
func (r *mapResolver) Resolve(key string) interface{} {
	val, found := r.values[key]
	if !found {
		return fmt.Errorf("no key '%s' in map", key)
	}
	return val
}

// Atomize is called when this object needs to be reduced to a primitive
func (r *mapResolver) Atomize() interface{} { return fmt.Sprintf("%s", r.values) }

var _ Atomizable = (*mapResolver)(nil)
var _ Resolvable = (*mapResolver)(nil)

// XType is an an enumeration of the possible types we can deal with
type XType int

// primitive types we convert to
const (
	XTypeNil = iota
	XTypeError
	XTypeString
	XTypeDecimal
	XTypeTime
	XTypeBool
	XTypeArray
)

// ToXAtom figures out the raw type of the passed in interface, returning that type
func ToXAtom(env Environment, val interface{}) (interface{}, XType, error) {
	if val == nil {
		return val, XTypeNil, nil
	}

	switch val := val.(type) {
	case error:
		return val, XTypeError, nil

	case string:
		return val, XTypeString, nil

	case decimal.Decimal:
		return val, XTypeDecimal, nil
	case int:
		return decimal.New(int64(val), 0), XTypeDecimal, nil

	case time.Time:
		return val, XTypeTime, nil

	case bool:
		return val, XTypeBool, nil

	case Array:
		return val, XTypeArray, nil
	}

	return val, XTypeNil, fmt.Errorf("Unknown type '%s' with value '%+v'", reflect.TypeOf(val), val)
}
