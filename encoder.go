package schema

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

var (
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	stringerType      = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
)

type encoderFunc func(reflect.Value) string

// Encoder encodes values from a struct into url.Values.
type Encoder struct {
	cache  *cache
	regenc map[reflect.Type]encoderFunc
}

// NewEncoder returns a new Encoder with defaults.
func NewEncoder() *Encoder {
	return &Encoder{cache: newCache(), regenc: make(map[reflect.Type]encoderFunc)}
}

// Encode encodes a struct into map[string][]string.
//
// Intended for use with url.Values.
func (e *Encoder) Encode(src interface{}, dst map[string][]string) error {
	v := reflect.ValueOf(src)

	return e.encode(v, dst)
}

// RegisterEncoder registers a converter for encoding a custom type.
func (e *Encoder) RegisterEncoder(value interface{}, encoder func(reflect.Value) string) {
	e.regenc[reflect.TypeOf(value)] = encoder
}

// SetAliasTag changes the tag used to locate custom field aliases.
// The default tag is "schema".
func (e *Encoder) SetAliasTag(tag string) {
	e.cache.tag = tag
}

// isValidStructPointer test if input value is a valid struct pointer.
func isValidStructPointer(v reflect.Value) bool {
	return v.Type().Kind() == reflect.Ptr && v.Elem().IsValid() && v.Elem().Type().Kind() == reflect.Struct
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func:
	case reflect.Map, reflect.Slice:
		return v.IsNil() || v.Len() == 0
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && isZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		type zero interface {
			IsZero() bool
		}
		if v.Type().Implements(reflect.TypeOf((*zero)(nil)).Elem()) {
			iz := v.MethodByName("IsZero").Call([]reflect.Value{})[0]
			return iz.Interface().(bool)
		}
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && isZero(v.Field(i))
		}
		return z
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}

func (e *Encoder) encode(v reflect.Value, dst map[string][]string) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return errors.New("schema: interface must be a struct")
	}
	t := v.Type()

	errors := MultiError{}

	for i := 0; i < v.NumField(); i++ {
		name, opts := fieldAlias(t.Field(i), e.cache.tag)
		if name == "-" {
			continue
		}

		// Encode struct pointer types if the field is a valid pointer and a struct.
		// Skip recursion when the pointer or the struct implements TextMarshaler or Stringer.
		if isValidStructPointer(v.Field(i)) && !e.hasCustomEncoder(v.Field(i).Type()) &&
			interfaceEncoder(v.Field(i).Type()) == nil &&
			interfaceEncoder(v.Field(i).Elem().Type()) == nil {
			err := e.encode(v.Field(i).Elem(), dst)
			if err != nil {
				errors[v.Field(i).Elem().Type().String()] = err
			}
			continue
		}

		encFunc := typeEncoder(v.Field(i).Type(), e.regenc)

		// Encode non-slice types and custom implementations immediately.
		if encFunc != nil {
			value := encFunc(v.Field(i))
			if opts.Contains("omitempty") && isZero(v.Field(i)) {
				continue
			}

			dst[name] = append(dst[name], value)
			continue
		}

		if v.Field(i).Type().Kind() == reflect.Struct {
			err := e.encode(v.Field(i), dst)
			if err != nil {
				errors[v.Field(i).Type().String()] = err
			}
			continue
		}

		if v.Field(i).Type().Kind() == reflect.Slice {
			encFunc = typeEncoder(v.Field(i).Type().Elem(), e.regenc)
		}

		if encFunc == nil {
			errors[v.Field(i).Type().String()] = fmt.Errorf("schema: encoder not found for %v", v.Field(i))
			continue
		}

		// Encode a slice.
		if v.Field(i).Len() == 0 && opts.Contains("omitempty") {
			continue
		}

		dst[name] = []string{}
		for j := 0; j < v.Field(i).Len(); j++ {
			dst[name] = append(dst[name], encFunc(v.Field(i).Index(j)))
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (e *Encoder) hasCustomEncoder(t reflect.Type) bool {
	_, exists := e.regenc[t]
	return exists
}

func typeEncoder(t reflect.Type, reg map[reflect.Type]encoderFunc) encoderFunc {
	if f, ok := reg[t]; ok {
		return f
	}
	if f := interfaceEncoder(t); f != nil {
		return f
	}

	switch t.Kind() {
	case reflect.Bool:
		return encodeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return encodeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return encodeUint
	case reflect.Float32:
		return encodeFloat32
	case reflect.Float64:
		return encodeFloat64
	case reflect.Ptr:
		f := typeEncoder(t.Elem(), reg)
		return func(v reflect.Value) string {
			if v.IsNil() {
				return "null"
			}
			return f(v.Elem())
		}
	case reflect.String:
		return encodeString
	default:
		return nil
	}
}

func interfaceEncoder(t reflect.Type) encoderFunc {
	if t == nil {
		return nil
	}
	if t.Implements(textMarshalerType) {
		return encodeTextMarshaler
	}
	if t.Kind() != reflect.Ptr && reflect.PointerTo(t).Implements(textMarshalerType) {
		return encodeTextMarshaler
	}
	if t.Implements(stringerType) {
		return encodeStringer
	}
	if t.Kind() != reflect.Ptr && reflect.PointerTo(t).Implements(stringerType) {
		return encodeStringer
	}
	return nil
}

func asInterface(v reflect.Value, iface reflect.Type) interface{} {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	if v.Type().Implements(iface) && v.CanInterface() {
		return v.Interface()
	}
	if v.Kind() != reflect.Ptr {
		if v.CanAddr() {
			p := v.Addr()
			if p.Type().Implements(iface) && p.CanInterface() {
				return p.Interface()
			}
		} else if v.CanInterface() {
			p := reflect.New(v.Type())
			p.Elem().Set(v)
			if p.Type().Implements(iface) {
				return p.Interface()
			}
		}
	}
	return nil
}

func encodeTextMarshaler(v reflect.Value) string {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return "null"
	}
	m, ok := asInterface(v, textMarshalerType).(encoding.TextMarshaler)
	if !ok || m == nil {
		return ""
	}
	b, err := m.MarshalText()
	if err != nil {
		return ""
	}
	return string(b)
}

func encodeStringer(v reflect.Value) string {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return "null"
	}
	s, ok := asInterface(v, stringerType).(fmt.Stringer)
	if !ok || s == nil {
		return ""
	}
	return s.String()
}

func encodeBool(v reflect.Value) string {
	return strconv.FormatBool(v.Bool())
}

func encodeInt(v reflect.Value) string {
	return strconv.FormatInt(int64(v.Int()), 10)
}

func encodeUint(v reflect.Value) string {
	return strconv.FormatUint(uint64(v.Uint()), 10)
}

func encodeFloat(v reflect.Value, bits int) string {
	return strconv.FormatFloat(v.Float(), 'f', 6, bits)
}

func encodeFloat32(v reflect.Value) string {
	return encodeFloat(v, 32)
}

func encodeFloat64(v reflect.Value) string {
	return encodeFloat(v, 64)
}

func encodeString(v reflect.Value) string {
	return v.String()
}
