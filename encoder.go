package schema

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type encoderFunc func(reflect.Value) string

// UrlValues represents url.Values which could be encoded with custom order.
type UrlValues struct {
	keys   []string
	values map[string][]string
}

// Values returns map[string][]string which can be used as url.Values.
func (v *UrlValues) Values() map[string][]string {
	return v.values
}

// Encode encodes the values into URL encoded form ("foo=quux&bar=baz") sorted by custom order.
func (v *UrlValues) Encode() string {
	if len(v.values) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, k := range v.keys {
		vs := v.values[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}

func (v *UrlValues) removeKey(key string) {
	for i, x := range v.keys {
		if x == key {
			v.keys = append(v.keys[:i], v.keys[i+1:]...)
			return
		}
	}
}

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
func (e *Encoder) Encode(src any, dst map[string][]string) error {
	v := reflect.ValueOf(src)
	values := &UrlValues{
		values: dst,
	}

	return e.encode(v, values)
}

// EncodeValues encodes a struct into UrlValues which will keep the order of the struct's fields.
func (e *Encoder) EncodeValues(src any) (*UrlValues, error) {
	v := reflect.ValueOf(src)
	values := &UrlValues{
		values: map[string][]string{},
	}

	if err := e.encode(v, values); err != nil {
		return nil, err
	}
	return values, nil
}

// RegisterEncoder registers a converter for encoding a custom type.
func (e *Encoder) RegisterEncoder(value any, encoder func(reflect.Value) string) {
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

func (e *Encoder) encode(v reflect.Value, values *UrlValues) error {
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
		if isValidStructPointer(v.Field(i)) && !e.hasCustomEncoder(v.Field(i).Type()) {
			err := e.encode(v.Field(i).Elem(), values)
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

			if _, ok := values.values[name]; !ok {
				values.keys = append(values.keys, name)
			}
			values.values[name] = append(values.values[name], value)
			continue
		}

		if v.Field(i).Type().Kind() == reflect.Struct {
			err := e.encode(v.Field(i), values)
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
		sliceLen := v.Field(i).Len()
		if sliceLen == 0 && opts.Contains("omitempty") {
			continue
		}

		if _, ok := values.values[name]; ok {
			values.removeKey(name)
		}
		values.keys = append(values.keys, name)
		values.values[name] = make([]string, 0, sliceLen)
		for j := 0; j < sliceLen; j++ {
			values.values[name] = append(values.values[name], encFunc(v.Field(i).Index(j)))
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
