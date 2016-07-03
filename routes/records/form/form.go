package form

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type (
	FormEncoder interface {
		FormEncode(name string) ([]byte, error)
	}

	FormDecoder interface {
		FormDecode(name string, values url.Values) error
	}
)

// Marshal converts a Go type to an HTML 5 form.
func Marshal(i interface{}) ([]byte, error) {
	if fe, ok := i.(FormEncoder); ok {
		return fe.FormEncode("")
	}

	return marshalValue("", reflect.ValueOf(i))
}

func marshalValue(name string, v reflect.Value) ([]byte, error) {
	if v.Type().Name() == "Time" {
		return encodeTime(name, v.Interface().(time.Time)), nil
	}

	switch v.Kind() {
	case reflect.Bool:
		return encodeBool(name, v.Bool()), nil
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		return encodeUint(name, v.Uint()), nil
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		return encodeInt(name, v.Int()), nil
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		return encodeFloat(name, v.Float()), nil
	case reflect.Ptr:
		return marshalValue(name, reflect.Indirect(v))
	case reflect.Slice:
		return marshalSlice(name, v)
	case reflect.String:
		return encodeString(name, v.String()), nil
	case reflect.Struct:
		return marshalStruct(name, v)
	case reflect.Interface:
		i := v.Interface()
		if fe, ok := i.(FormEncoder); ok {
			return fe.FormEncode(name)
		}

		return marshalValue(name, reflect.ValueOf(i))
	default:
		return nil, nil
	}
}

func marshalPrimitive(name string, i interface{}) (template.HTML, error) {
	b, err := marshalPrimitiveBytes(name, i)
	return template.HTML(string(b)), err
}

func marshalPrimitiveBytes(name string, i interface{}) ([]byte, error) {
	if i == nil {
		return nil, nil
	}

	v := reflect.ValueOf(i)
	if v.Type().Name() == "Time" {
		return encodeTime(name, v.Interface().(time.Time)), nil
	}

	switch v.Kind() {
	case reflect.Bool:
		return encodeBool(name, v.Bool()), nil
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		return encodeUint(name, v.Uint()), nil
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		return encodeInt(name, v.Int()), nil
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		return encodeFloat(name, v.Float()), nil
	case reflect.Ptr:
		return marshalPrimitiveBytes(name, reflect.Indirect(v))
	case reflect.String:
		return encodeString(name, v.String()), nil
	default:
		return nil, nil
	}
}

func encodeBool(name string, quote bool) []byte {
	checked := "checked"
	if !quote {
		checked = ""
	}
	return []byte(fmt.Sprintf(`<input name="%s" type="checkbox" %s/>`, name, checked))
}

func encodeInt(name string, quote int64) []byte {
	return []byte(fmt.Sprintf(`<input name="%s" type="number" value="%d" />`, name, quote))
}

func encodeUint(name string, quote uint64) []byte {
	return []byte(fmt.Sprintf(`<input name="%s" type="number" value="%d" />`, name, quote))
}

func encodeFloat(name string, quote float64) []byte {
	return []byte(fmt.Sprintf(`<input name="%s" type="number" value="%f" />`, name, quote))
}

func encodeTime(name string, quote time.Time) []byte {
	return []byte(fmt.Sprintf(`<input name="%s" type="datetime-local" value="%s" />`, name, quote.Format(time.RFC3339)))
}

var sliceTemplate = template.Must(template.New("main").Funcs(template.FuncMap{
	"dict":             Dict,
	"marshalPrimitive": marshalPrimitive,
}).ParseFiles("./slice.tmpl"))

// Dict constructs a map out of the sequential key value pairs provided,
// Used to construct custom context while in a template. i.e.,
// if Dict was defined in the funcMap to be "dict"
// {{template "CallThisTemplate" dict "user" $user "routes" .Data.Routes}}
// now the CallThisTemplate template  gets a context with .user and .routes defined
var Dict = func(vals ...interface{}) (map[string]interface{}, error) {
	if len(vals)%2 != 0 {
		return nil, errors.New("Must pass element pairs")
	}

	dict := make(map[string]interface{}, len(vals)/2)

	for i := 0; i < len(vals); i += 2 {
		key, ok := vals[i].(string)
		if !ok {
			return nil, errors.New("Keys must be strings")
		}

		dict[key] = vals[i+1]
	}

	return dict, nil
}

func marshalSlice(name string, v reflect.Value) ([]byte, error) {
	b := new(bytes.Buffer)
	if err := sliceTemplate.Execute(b, map[string]interface{}{
		"Name":   name,
		"name":   template.JS(strings.Replace(strings.Replace(name, "[", "_", -1), "]", "_", -1)),
		"values": v.Interface(),
	}); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func encodeString(name, quote string) []byte {
	return []byte(fmt.Sprintf(`<input name="%s" type="text" value="%s" />`, name, quote))
}

func marshalStruct(prefix string, s reflect.Value) ([]byte, error) {
	t := s.Type()
	b := new(bytes.Buffer)
	for i := 0; i < t.NumField(); i++ {
		bytes, err := marshalField(prefix, s, t.Field(i))
		if err != nil {
			return nil, err
		}
		if _, err := b.Write(bytes); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}

func marshalField(prefix string, s reflect.Value, f reflect.StructField) ([]byte, error) {
	return marshalValue(prepend(prefix, f.Name), s.FieldByIndex(f.Index))
}

func prepend(prefix, name string) string {
	if prefix == "" {
		return name
	}
	if name == "" {
		return prefix
	}
	return fmt.Sprintf("%s[%s]", prefix, name)
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
		return marshalSlice(name, v)
	case reflect.String:
		return encodeString(name, v.String()), nil
	case reflect.Struct:
		return marshalStruct(name, v)
	default:
		panic(fmt.Sprintf("unrecognized kind: %s, of value: %v", v.Kind(), v))
	}
}
