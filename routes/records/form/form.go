package form

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"time"

	"github.com/elos/x/html"
)

type (
	// Marshaler overrides the default implementation of marshalling a go type
	// into an HTML5 form. The parameterization of the namespace must be respected
	// in HTML element naming as well as in field naming.
	Marshaler interface {
		MarshalForm(namespace string) ([]byte, error)
	}

	// Unmarshaler overrides the default implementation of unmarshalling a go type
	// from url.Values of a HTML form. The parametrization namespace is symetric and
	// guaranteed to be the same as that given during marshalling.
	Unmarshaler interface {
		UnmarshalForm(values url.Values, namespace string) error
	}
)

func Must(bs []byte, err error) []byte {
	if err != nil {
		panic(fmt.Sprintf("err != nil: %v", err))
	}
	return bs
}

// --- Marshal {{{

// Marshal marshals a Go type into an HTML5 form.
func Marshal(i interface{}, namespace string) ([]byte, error) {
	// Protect against an immediate nil.
	if i == nil {
		return nil, nil
	}

	// We are asserting on the interface at a semantic level.
	if fe, ok := i.(Marshaler); ok {
		return fe.MarshalForm(namespace)
	}

	return marshalValue(namespace, reflect.ValueOf(i))
}

// marshalValue is the internal implementation, you will see the
// named implementations, then the various reflection matches.
func marshalValue(namespace string, v reflect.Value) ([]byte, error) {
	if m, ok := v.Interface().(Marshaler); ok {
		return m.MarshalForm(namespace)
	}

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
		return marshalSlice(namespace, v)
	case reflect.Map:
		return marshalComposite(namespace, v)
	case reflect.Struct:
		return marshalStruct(namespace, v)

	// Indirects
	case reflect.Ptr:
		if v.IsNil() {
			return nil, nil
		}

		return marshalValue(namespace, reflect.Indirect(v))
	case reflect.Interface:
		if v.IsNil() {
			return nil, nil
		}

		i := v.Interface()
		if fe, ok := i.(Marshaler); ok {
			return fe.MarshalForm(namespace)
		}

		return marshalValue(namespace, reflect.ValueOf(i))

	// The default case is an error, a failure to evaluate the
	// type of the value and select an adequate marshalling.
	default:
		return nil, fmt.Errorf("unable to marshal value %v", v)
	}
}

func encodeTime(name string, quote time.Time) []byte {
	return append(html.Must((&html.Label{
		For:   name,
		Label: path.Base(name),
	}).MarshalText()), html.Must((&html.Input{
		Name:  name,
		Type:  html.Input_DATETIME_LOCAL,
		Value: quote.Format(time.RFC3339),
	}).MarshalText())...)
}

func encodeBool(name string, quote bool) []byte {
	return append(html.Must((&html.Label{
		For:   name,
		Label: path.Base(name),
	}).MarshalText()), html.Must((&html.Input{
		Name:    name,
		Type:    html.Input_CHECKBOX,
		Checked: quote,
	}).MarshalText())...)
}

func encodeInt(name string, quote int64) []byte {
	return append(html.Must((&html.Label{
		For:   name,
		Label: path.Base(name),
	}).MarshalText()), html.Must((&html.Input{
		Type:  html.Input_NUMBER,
		Name:  name,
		Value: fmt.Sprintf("%d", quote),
	}).MarshalText())...)
}

func encodeUint(name string, quote uint64) []byte {
	return append(html.Must((&html.Label{
		For:   name,
		Label: path.Base(name),
	}).MarshalText()), html.Must((&html.Input{
		Type:  html.Input_NUMBER,
		Name:  name,
		Value: fmt.Sprintf("%d", quote),
	}).MarshalText())...)
}

func encodeFloat(name string, quote float64) []byte {
	return append(html.Must((&html.Label{
		For:   name,
		Label: path.Base(name),
	}).MarshalText()), html.Must((&html.Input{
		Type:  html.Input_NUMBER,
		Name:  name,
		Value: fmt.Sprintf("%f", quote),
	}).MarshalText())...)
}

func encodeString(name, quote string) []byte {
	return append(html.Must((&html.Label{
		For:   name,
		Label: path.Base(name),
	}).MarshalText()), html.Must((&html.Input{
		Type:  html.Input_TEXT,
		Name:  name,
		Value: quote,
	}).MarshalText())...)
}

func marshalSlice(namespace string, s reflect.Value) ([]byte, error) {
	b := new(bytes.Buffer)
	_, err := b.WriteString(fmt.Sprintf(`<fieldset id="%s" class="expandable" data-count="%d"><legend>%s</legend>`, namespace, s.Len(), path.Base(namespace)))
	if err != nil {
		return nil, err
	}

	b.WriteString(fmt.Sprintf(`<div style="display:none" id="%s">`, path.Join(namespace, "zero")))
	bs, err := marshalElem(namespace, reflect.Zero(s.Type().Elem()), -1)
	if err != nil {
		return nil, err
	}
	if _, err := b.Write(bs); err != nil {
		return nil, err
	}
	b.WriteString("</div>")
	for i := 0; i < s.Len(); i++ {
		bytes, err := marshalElem(namespace, s.Index(i), i)
		if err != nil {
			return nil, err
		}
		if _, err := b.Write(bytes); err != nil {
			return nil, err
		}
		if len(bytes) != 0 {
			if _, err := b.WriteString("<br>"); err != nil {
				return nil, err
			}
		}
	}
	_, err = b.WriteString("</fieldset>")
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func marshalElem(namespace string, s reflect.Value, i int) ([]byte, error) {
	return marshalValue(path.Join(namespace, strconv.Itoa(i)), s)
}

func marshalComposite(name string, v reflect.Value) ([]byte, error) {
	bytes, err := json.MarshalIndent(v.Interface(), "", "	")
	if err != nil {
		return nil, err
	}

	return append(html.Must((&html.Label{
		For:   name,
		Label: path.Base(name),
	}).MarshalText()), html.Must((&html.TextArea{
		Name:    name,
		Content: string(bytes),
	}).MarshalText())...), nil
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
		if len(bytes) != 0 {
			if _, err := b.WriteString("<br>"); err != nil {
				return nil, err
			}
		}
	}
	_, err = b.WriteString("</fieldset>")
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func marshalField(namespace string, s reflect.Value, f reflect.StructField) ([]byte, error) {
	switch field := s.FieldByIndex(f.Index); f.PkgPath {
	case "": // exported
		return marshalValue(path.Join(namespace, f.Name), field)
	default: // unexported
		return nil, nil
	}
}

// --- }}}

func Unmarshal(form url.Values, i interface{}, namespace string) error {
	// Protect against an immediate nil.
	if i == nil {
		return nil
	}

	// We are asserting on the interface at a semantic level.
	if fe, ok := i.(Unmarshaler); ok {
		return fe.UnmarshalForm(form, namespace)
	}

	switch v := reflect.ValueOf(i); v.Kind() {
	default:
		if v := reflect.ValueOf(i); v.Kind() != reflect.Ptr {
			return fmt.Errorf("should be a pointer value")
		}

	// Composite types
	// case reflect.Slice: doesn't work
	case reflect.Map:
	}

	return unmarshalValue(form, reflect.ValueOf(i), namespace)
}

func CantSet(v reflect.Value) error {
	return fmt.Errorf("can not set value: %v", v)
}

func unmarshalValue(form url.Values, v reflect.Value, namespace string) error {
	if v.Type().Name() == "Time" {
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeTime(param, v)
			}
		}
	}

	i := v.Interface()
	if fe, ok := i.(Unmarshaler); ok {
		return fe.UnmarshalForm(form, namespace)
	}

	switch v.Kind() {

	// Primitives
	case reflect.Bool:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeBool(param, v)
			}
		}
	case reflect.Uint8:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeUint(param, v, 8)
			}
		}
	case reflect.Uint16:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeUint(param, v, 16)
			}
		}
	case reflect.Uint32:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeUint(param, v, 32)
			}
		}
	case reflect.Uint:
		fallthrough
	case reflect.Uint64:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeUint(param, v, 64)
			}
		}
	case reflect.Int8:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeInt(param, v, 8)
			}
		}
	case reflect.Int16:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeInt(param, v, 16)
			}
		}
	case reflect.Int32:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeInt(param, v, 32)
			}
		}
	case reflect.Int:
		fallthrough
	case reflect.Int64:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeInt(param, v, 64)
			}
		}
	case reflect.Float32:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeFloat(param, v, 32)
			}
		}
	case reflect.Float64:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return decodeFloat(param, v, 64)
			}
		}
	case reflect.String:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				v.SetString(param)
			}
			return nil
		}

	// Composites
	case reflect.Slice:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			switch param := form.Get(namespace); param {
			case "":
				return nil
			default:
				return unmarshalSlice(param, v)
			}
		}
	case reflect.Map:
		switch param := form.Get(namespace); param {
		case "":
			return nil
		default:
			return unmarshalMap(param, v)
		}
	case reflect.Struct:
		switch v.CanSet() {
		case false:
			return CantSet(v)
		default:
			return unmarshalStruct(form, v, namespace)
		}

	// Indirects
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		return unmarshalValue(form, reflect.Indirect(v), namespace)
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}

		return unmarshalValue(form, v, namespace)

	// The default case is an error, a failure to evaluate the
	// type of the value and select an adequate marshalling.
	default:
		return fmt.Errorf("unable to unmarshal value %v", v)
	}
}

const datetimeLocalLayout = "2006-01-02T15:04"

func decodeTime(param string, v reflect.Value) error {
	t, err := time.Parse(datetimeLocalLayout, param)
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(t))
	return nil
}

func decodeBool(param string, v reflect.Value) error {
	b, err := strconv.ParseBool(param)
	if err != nil {
		return err
	}

	v.SetBool(b)
	return nil
}

func decodeUint(param string, v reflect.Value, bitSize int) error {
	n, err := strconv.ParseUint(param, 10, bitSize)
	if err != nil {
		return err
	}

	v.SetUint(n)
	return nil
}

func decodeInt(param string, v reflect.Value, bitSize int) error {
	n, err := strconv.ParseInt(param, 10, bitSize)
	if err != nil {
		return err
	}

	v.SetInt(n)
	return nil
}

func decodeFloat(param string, v reflect.Value, bitSize int) error {
	f, err := strconv.ParseFloat(param, bitSize)
	if err != nil {
		return err
	}

	v.SetFloat(f)
	return nil
}

func unmarshalSlice(param string, v reflect.Value) error {
	// Create a pointer to a composite value and set it to the composite
	x := reflect.New(v.Type())
	x.Elem().Set(reflect.MakeSlice(v.Type(), 0, 0))

	if err := json.Unmarshal([]byte(param), x.Interface()); err != nil {
		return err
	}

	v.Set(x.Elem())
	return nil
}

func unmarshalMap(param string, v reflect.Value) error {
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}

	x := reflect.New(v.Type())
	x.Elem().Set(v)

	return json.Unmarshal([]byte(param), x.Interface())
}

func unmarshalStruct(form url.Values, s reflect.Value, namespace string) error {
	t := s.Type()

	for i := 0; i < t.NumField(); i++ {
		if err := unmarshalField(form, namespace, s, t.Field(i)); err != nil {
			return err
		}
	}

	return nil
}

func unmarshalField(form url.Values, namespace string, s reflect.Value, f reflect.StructField) error {
	switch field := s.FieldByIndex(f.Index); field.CanSet() {
	case false: // unexported
		return nil
	default:
		return unmarshalValue(form, field, path.Join(namespace, f.Name))
	}
}
