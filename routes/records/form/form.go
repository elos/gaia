package form

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"time"
)

type (
	// FormMarshaler overrides the default implementation of marshalling a go type
	// into an HTML5 form. The parameterization of the namespace must be respected
	// in HTML element naming as well as in field naming.
	FormMarshaler interface {
		FormMarshal(namespace string) ([]byte, error)
	}

	// FormUnmarshaler overrides the default implementation of unmarshalling a go type
	// from url.Values of a HTML form. The parametrization namespace is symetric and
	// guaranteed to be the same as that given during marshalling.
	FormUnmarshaler interface {
		FormUnmarshal(namespace string, values url.Values) error
	}
)

// Marshal marshals a Go type into an HTML5 form.
func Marshal(i interface{}, namespace string) ([]byte, error) {
	// Protect against an immediate nil.
	if i == nil {
		return nil, nil
	}

	// We are asserting on the interface at a semantic level.
	if fe, ok := i.(FormMarshaler); ok {
		return fe.FormMarshal(namespace)
	}

	return marshalValue(namespace, reflect.ValueOf(i))
}

// marshalValue is the internal implementation, you will see the
// named implementations, then the various reflection matches.
func marshalValue(namespace string, v reflect.Value) ([]byte, error) {
	if v.Type().Name() == "Time" {
		return encodeTime(namespace, v.Interface().(time.Time)), nil
	}

	switch v.Kind() {

	// Primitives
	case reflect.Bool:
		return encodeBool(namespace, v.Bool()), nil
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		return encodeUint(namespace, v.Uint()), nil
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		return encodeInt(namespace, v.Int()), nil
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		return encodeFloat(namespace, v.Float()), nil
	case reflect.String:
		return encodeString(namespace, v.String()), nil

	// Composites
	case reflect.Slice:
		fallthrough
	case reflect.Map:
		return marshalComposite(namespace, v)
	case reflect.Struct:
		return marshalStruct(namespace, v)

	// Indirects
	case reflect.Ptr:
		return marshalValue(namespace, reflect.Indirect(v))
	case reflect.Interface:
		i := v.Interface()
		if fe, ok := i.(FormMarshaler); ok {
			return fe.FormMarshal(namespace)
		}

		return marshalValue(namespace, reflect.ValueOf(i))

	// The default case is an error, a failure to evaluate the
	// type of the value and select an adequate marshalling.
	default:
		return nil, fmt.Errorf("unable to marshal value %v", v)
	}
}

func encodeTime(name string, quote time.Time) []byte {
	return []byte(fmt.Sprintf(`<label for="%s">%s</label><input name="%s" type="datetime-local" value="%s" />`, name, path.Base(name), name, quote.Format(time.RFC3339)))
}

func encodeBool(name string, quote bool) []byte {
	checked := "checked"
	if !quote {
		checked = ""
	}
	return []byte(fmt.Sprintf(`<label for="%s">%s</label><input name="%s" type="checkbox" %s/>`, name, path.Base(name), name, checked))
}

func encodeInt(name string, quote int64) []byte {
	return []byte(fmt.Sprintf(`<label for="%s">%s</label><input name="%s" type="number" value="%d" />`, name, path.Base(name), name, quote))
}

func encodeUint(name string, quote uint64) []byte {
	return []byte(fmt.Sprintf(`<label for="%s">%s</label><input name="%s" type="number" value="%d" />`, name, path.Base(name), name, quote))
}

func encodeFloat(name string, quote float64) []byte {
	return []byte(fmt.Sprintf(`<label for="%s">%s</label><input name="%s" type="number" value="%f" />`, name, path.Base(name), name, quote))
}

func encodeString(name, quote string) []byte {
	return []byte(fmt.Sprintf(`<label for="%s">%s</label><input name="%s" type="text" value="%s" />`, name, path.Base(name), name, quote))
}

func marshalComposite(name string, v reflect.Value) ([]byte, error) {
	bytes, err := json.MarshalIndent(v.Interface(), "", "	")
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf(`<label for="%s">%s</label><textarea name="%s">%s</textarea>`, name, path.Base(name), name, string(bytes))), nil
}

func marshalStruct(namespace string, s reflect.Value) ([]byte, error) {
	t := s.Type()
	b := new(bytes.Buffer)
	_, err := b.WriteString(fmt.Sprintf("<fieldset><legend>%s</legend>", path.Base(namespace)))
	if err != nil {
		return nil, err
	}
	for i := 0; i < t.NumField(); i++ {
		bytes, err := marshalField(namespace, s, t.Field(i))
		if err != nil {
			return nil, err
		}
		if _, err := b.Write(bytes); err != nil {
			return nil, err
		}
		if _, err := b.WriteString("<br>"); err != nil {
			return nil, err
		}
	}
	_, err = b.WriteString("</fieldset>")
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func marshalField(namespace string, s reflect.Value, f reflect.StructField) ([]byte, error) {
	return marshalValue(path.Join(namespace, f.Name), s.FieldByIndex(f.Index))
}

func Unmarshal(form url.Values, i interface{}) error {
	return nil
}

func unmarshalValue(name string, v reflect.Value) ([]byte, error) {
	if v.Type().Name() == "Time" {
		//return decodeTime(name, v.Interface().(time.Time)), nil
	}

	switch v.Kind() {
	case reflect.Bool:
		return encodeBool(name, v.Bool()), nil
	case reflect.Int:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		return encodeInt(name, v.Int()), nil
	case reflect.Float64:
		return encodeFloat(name, v.Float()), nil
	case reflect.Ptr:
		return marshalValue(name, reflect.Indirect(v))
	case reflect.Slice:
		fallthrough
	case reflect.Map:
		return marshalComposite(name, v)
	case reflect.String:
		return encodeString(name, v.String()), nil
	case reflect.Struct:
		return marshalStruct(name, v)
	default:
		panic(fmt.Sprintf("unrecognized kind: %s, of value: %v", v.Kind(), v))
	}
}
