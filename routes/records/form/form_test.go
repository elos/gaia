package form_test

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes/records/form"
	"github.com/elos/models/user"
)

type NamedPrimitive int

func (n NamedPrimitive) MarshalForm(ns string) ([]byte, error) {
	return []byte("barfoobar"), nil
}

type Composite struct {
	N NamedPrimitive
}

// --- TestMarshal {{{
func TestMarshal(t *testing.T) {
	cases := []struct {
		name      string
		structure interface{}
		output    string
	}{
		// Named
		// time.Time
		{
			name:      "time.Time",
			structure: time.Unix(1136214245, 0),
			output:    `<label for="time.Time">time.Time</label><input name="time.Time" type="datetime-local" value="2006-01-02T07:04:05-08:00" />`,
		},
		// Primitives
		// Byte (reflect.Kind == Uint8)
		{
			name:      "byte",
			structure: byte(2),
			output:    `<label for="byte">byte</label><input name="byte" type="number" value="2" />`,
		},
		// Bool
		{
			name:      "bool_true",
			structure: true,
			output:    `<label for="bool_true">bool_true</label><input name="bool_true" type="checkbox" checked />`,
		},
		{
			name:      "bool_false",
			structure: false,
			output:    `<label for="bool_false">bool_false</label><input name="bool_false" type="checkbox" />`,
		},
		// Int
		{
			name:      "int",
			structure: int(5),
			output:    `<label for="int">int</label><input name="int" type="number" value="5" />`,
		},
		{
			name:      "int8",
			structure: int8(5),
			output:    `<label for="int8">int8</label><input name="int8" type="number" value="5" />`,
		},
		{
			name:      "int16",
			structure: int16(5),
			output:    `<label for="int16">int16</label><input name="int16" type="number" value="5" />`,
		},
		{
			name:      "int32",
			structure: int32(5),
			output:    `<label for="int32">int32</label><input name="int32" type="number" value="5" />`,
		},
		{
			name:      "int64",
			structure: int64(5),
			output:    `<label for="int64">int64</label><input name="int64" type="number" value="5" />`,
		},
		// Uint
		{
			name:      "uint",
			structure: uint(5),
			output:    `<label for="uint">uint</label><input name="uint" type="number" value="5" />`,
		},
		{
			name:      "uint8",
			structure: uint8(5),
			output:    `<label for="uint8">uint8</label><input name="uint8" type="number" value="5" />`,
		},
		{
			name:      "uint16",
			structure: uint16(5),
			output:    `<label for="uint16">uint16</label><input name="uint16" type="number" value="5" />`,
		},
		{
			name:      "uint32",
			structure: uint32(5),
			output:    `<label for="uint32">uint32</label><input name="uint32" type="number" value="5" />`,
		},
		{
			name:      "uint64",
			structure: uint64(5),
			output:    `<label for="uint64">uint64</label><input name="uint64" type="number" value="5" />`,
		},
		// Floats
		{
			name:      "float32",
			structure: float32(123.0),
			output:    `<label for="float32">float32</label><input name="float32" type="number" value="123.000000" />`,
		},
		{
			name:      "float64",
			structure: float32(123.0),
			output:    `<label for="float64">float64</label><input name="float64" type="number" value="123.000000" />`,
		},
		// String
		{
			name:      "string",
			structure: "here is a string",
			output:    `<label for="string">string</label><input name="string" type="text" value="here is a string" />`,
		},
		// Interface
		{
			name:      "interface{}(string)",
			structure: interface{}("string"),
			output:    `<label for="interface{}(string)">interface{}(string)</label><input name="interface{}(string)" type="text" value="string" />`,
		},

		// Composites
		// Slices
		{
			name:      "[]int",
			structure: []int{1, 2, 3},
			output: strings.TrimSpace(`
<fieldset class="slice-fieldset"><legend>[]int</legend><div class="zero-value"><label for="[]int/%index%">%index%</label><input name="[]int/%index%" type="number" value="0" /></div><label for="[]int/0">0</label><input name="[]int/0" type="number" value="1" /><br><label for="[]int/1">1</label><input name="[]int/1" type="number" value="2" /><br><label for="[]int/2">2</label><input name="[]int/2" type="number" value="3" /><br></fieldset>
			`),
		},
		{
			name:      "[]string",
			structure: []string{"foo", "bar", "tod"},
			output: strings.TrimSpace(`
<fieldset class="slice-fieldset"><legend>[]string</legend><div class="zero-value"><label for="[]string/%index%">%index%</label><input name="[]string/%index%" type="text" /></div><label for="[]string/0">0</label><input name="[]string/0" type="text" value="foo" /><br><label for="[]string/1">1</label><input name="[]string/1" type="text" value="bar" /><br><label for="[]string/2">2</label><input name="[]string/2" type="text" value="tod" /><br></fieldset>
			`),
		},
		{
			name:      "[]byte",
			structure: []byte{1, 2, 3, 4, 5},
			output: strings.TrimSpace(`
<fieldset class="slice-fieldset"><legend>[]byte</legend><div class="zero-value"><label for="[]byte/%index%">%index%</label><input name="[]byte/%index%" type="number" value="0" /></div><label for="[]byte/0">0</label><input name="[]byte/0" type="number" value="1" /><br><label for="[]byte/1">1</label><input name="[]byte/1" type="number" value="2" /><br><label for="[]byte/2">2</label><input name="[]byte/2" type="number" value="3" /><br><label for="[]byte/3">3</label><input name="[]byte/3" type="number" value="4" /><br><label for="[]byte/4">4</label><input name="[]byte/4" type="number" value="5" /><br></fieldset>
            `),
		},
		{
			name: "[]structure",
			structure: []struct {
			    I []int
			}{
				struct{ I []int }{
					I: []int{5},
				},
			},
			output: strings.TrimSpace(`
<fieldset class="slice-fieldset"><legend>[]structure</legend><div class="zero-value"><fieldset><legend>%index%</legend><fieldset class="slice-fieldset"><legend>I</legend><div class="zero-value"><label for="[]structure/%index%/I/%index%">%index%</label><input name="[]structure/%index%/I/%index%" type="number" value="0" /></div></fieldset><br></fieldset></div><fieldset><legend>0</legend><fieldset class="slice-fieldset"><legend>I</legend><div class="zero-value"><label for="[]structure/0/I/%index%">%index%</label><input name="[]structure/0/I/%index%" type="number" value="0" /></div><label for="[]structure/0/I/0">0</label><input name="[]structure/0/I/0" type="number" value="5" /><br></fieldset><br></fieldset><br></fieldset>
            `),
		},
		// Maps
		{
			name: "map[string]interface{}",
			structure: map[string]interface{}{
				"this": map[string]interface{}{
					"is": 1,
					"json": map[string]interface{}{
						"crazy": "stuff",
					},
				},
			},
			output: strings.TrimSpace(`
<label for="map[string]interface{}">map[string]interface{}</label><textarea name="map[string]interface{}">{
	"this": {
		"is": 1,
		"json": {
			"crazy": "stuff"
		}
	}
}</textarea>
			`),
		},

		// Structures
		{
			name: "bool_field_true",
			structure: struct {
				B bool
			}{
				B: true,
			},
			output: `<fieldset><legend>bool_field_true</legend><label for="bool_field_true/B">B</label><input name="bool_field_true/B" type="checkbox" checked /><br></fieldset>`,
		},
		{
			name: "bool_field_false",
			structure: struct {
				B bool
			}{
				B: false,
			},
			output: `<fieldset><legend>bool_field_false</legend><label for="bool_field_false/B">B</label><input name="bool_field_false/B" type="checkbox" /><br></fieldset>`,
		},
		{
			name: "integer_field",
			structure: struct {
				I int
			}{
				I: 45,
			},
			output: `<fieldset><legend>integer_field</legend><label for="integer_field/I">I</label><input name="integer_field/I" type="number" value="45" /><br></fieldset>`,
		},
		{
			name: "float_field",
			structure: struct {
				F float64
			}{
				F: 54.3,
			},
			output: `<fieldset><legend>float_field</legend><label for="float_field/F">F</label><input name="float_field/F" type="number" value="54.300000" /><br></fieldset>`,
		},
		{
			name: "string_field",
			structure: struct {
				S string
			}{
				S: "foo bar",
			},
			output: `<fieldset><legend>string_field</legend><label for="string_field/S">S</label><input name="string_field/S" type="text" value="foo bar" /><br></fieldset>`,
		},
		{
			name: "nested_structure",
			structure: struct {
				S struct {
					I int
				}
			}{
				S: struct{ I int }{
					I: 5,
				},
			},
			output: `<fieldset><legend>nested_structure</legend><fieldset><legend>S</legend><label for="nested_structure/S/I">I</label><input name="nested_structure/S/I" type="number" value="5" /><br></fieldset><br></fieldset>`,
		},
		{
			name: "ignore_unexported",
			structure: struct {
				unexp int
			}{
				unexp: 5,
			},
			output: `<fieldset><legend>ignore_unexported</legend></fieldset>`,
		},
		{
			name: "field override",
			structure: &Composite{
				N: NamedPrimitive(1),
			},
			output: `<fieldset><legend>field override</legend>barfoobar<br></fieldset>`,
		},

		// Form struct
		{
			name: "form_struct",
			structure: &form.Form{
				Action: "/action/",
				Method: "post",
			},
			output: `<form action="/action/" method="post"></form>`,
		},
		{
			name: "form_struct/with_value",
			structure: &form.Form{
				Action: "/action/",
				Method: "post",
				Value: struct {
					Foo string
					Bar int
				}{
					Foo: "foo",
					Bar: 8,
				},
				Name: "Structure",
			},
			output: `<form action="/action/" method="post" name="Structure"><fieldset><legend>Structure</legend><label for="Structure/Foo">Foo</label><input name="Structure/Foo" type="text" value="foo" /><br><label for="Structure/Bar">Bar</label><input name="Structure/Bar" type="number" value="8" /><br></fieldset></form>`,
		},

		// Nullity
		{
			name: "nil_pointer",
			structure: struct {
				S *struct {
					val string
				}
			}{
				S: nil,
			},
			output: `<fieldset><legend>nil_pointer</legend></fieldset>`,
		},
		{
			name: "nil_interface",
			structure: struct {
				Val interface{}
			}{
				Val: nil,
			},
			output: `<fieldset><legend>nil_interface</legend></fieldset>`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bytes, err := form.Marshal(c.structure, c.name)
			if err != nil {
				t.Fatalf("form.Marshal error: %v", err)
			}

			out := string(bytes)
			if got, want := out, c.output; got != want {
				t.Errorf("output: got,\n%s\nwant,\n%s", got, want)
			}
		})
	}
}

// --- }}}

func TestMarshalTask(t *testing.T) {
	t.Skip()
	db := mem.NewDB()
	user, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create error: %v", err)
	}

	bytes, err := form.Marshal(user, "user")
	if err != nil {
		t.Fatalf("form.Marshal error: %v", err)
	}

	if got, want := string(bytes), ""; got != want {
		t.Fatalf("form.Marshal: got\n%s\nwant,\n%s", got, want)
	}
}

func TestUnmarshal(t *testing.T) {
	cases := []struct {
		name   string
		values url.Values
		// into should always be a pointer
		into interface{}
		// want should always be the dereferenced type of into
		want interface{}
	}{
		// Named
		{
			name: "time",
			values: url.Values{
				"time": []string{"2016-07-08T14:04"},
			},
			into: new(time.Time),
			want: time.Date(2016, 7, 8, 14, 4, 0, 0, time.UTC),
		},

		// Primitives
		{
			name: "bool",
			values: url.Values{
				"bool": []string{"true"},
			},
			into: new(bool),
			want: true,
		},
		{
			name: "byte",
			values: url.Values{
				"byte": []string{"2"},
			},
			into: new(byte),
			want: byte(2),
		},
		{
			name: "uint8",
			values: url.Values{
				"uint8": []string{"8"},
			},
			into: new(uint8),
			want: uint8(8),
		},
		{
			name: "uint16",
			values: url.Values{
				"uint16": []string{"16"},
			},
			into: new(uint16),
			want: uint16(16),
		},
		{
			name: "uint32",
			values: url.Values{
				"uint32": []string{"32"},
			},
			into: new(uint32),
			want: uint32(32),
		},
		{
			name: "uint",
			values: url.Values{
				"uint": []string{"54"},
			},
			into: new(uint),
			want: uint(54),
		},
		{
			name: "uint64",
			values: url.Values{
				"uint64": []string{"64"},
			},
			into: new(uint64),
			want: uint64(64),
		},
		{
			name: "int8",
			values: url.Values{
				"int8": []string{"-8"},
			},
			into: new(int8),
			want: int8(-8),
		},
		{
			name: "int16",
			values: url.Values{
				"int16": []string{"-16"},
			},
			into: new(int16),
			want: int16(-16),
		},
		{
			name: "int32",
			values: url.Values{
				"int32": []string{"-32"},
			},
			into: new(int32),
			want: int32(-32),
		},
		{
			name: "int",
			values: url.Values{
				"int": []string{"-54"},
			},
			into: new(int),
			want: int(-54),
		},
		{
			name: "int64",
			values: url.Values{
				"int64": []string{"-64"},
			},
			into: new(int64),
			want: int64(-64),
		},
		{
			name: "float32",
			values: url.Values{
				"float32": []string{"32.32"},
			},
			into: new(float32),
			want: float32(32.32),
		},
		{
			name: "float64",
			values: url.Values{
				"float64": []string{"-64.64"},
			},
			into: new(float64),
			want: float64(-64.64),
		},
		{
			name: "string",
			values: url.Values{
				"string": []string{"this is a string"},
			},
			into: new(string),
			want: "this is a string",
		},

		// Composites
		/*
				{
					name: "[]int",
					values: url.Values{
						"[]int": []string{"[1,2,3]"},
					},
					into: make([]int, 0),
					want: []int{1, 2, 3},
				},
			{
				name: "[]byte",
				values: url.Values{
					"[]byte": []string{"AQIDBAU="},
				},
				into: make([]byte, 0),
				want: []byte{1, 2, 3, 4, 5},
			},
		*/
		{
			name: "map[string]int",
			values: url.Values{
				"map[string]int": []string{`{"foo": 1, "bar": 2}`},
			},
			into: make(map[string]int),
			want: map[string]int{
				"foo": 1,
				"bar": 2,
			},
		},

		// Structures
		{
			name: "struct/bool",
			values: url.Values{
				"struct/bool/B": []string{"true"},
			},
			into: new(struct {
				B bool
			}),
			want: struct {
				B bool
			}{
				B: true,
			},
		},
		{
			name: "struct/composite",
			values: url.Values{
				"struct/composite/Foo": []string{`["one", "two"]`},
				"struct/composite/Bar": []string{`{"1": 1, "2": 2}`},
			},
			into: new(struct {
				Foo []string
				Bar map[string]int
			}),
			want: struct {
				Foo []string
				Bar map[string]int
			}{
				Foo: []string{"one", "two"},
				Bar: map[string]int{
					"1": 1,
					"2": 2,
				},
			},
		},
		{
			name: "struct/alreadystate",
			values: url.Values{
				"struct/composite/Bar": []string{`null`},
			},
			into: &struct {
				Foo string
				Bar map[string]string
			}{
				Foo: "don't modify",
				Bar: map[string]string{
					"don't": "modify",
				},
			},
			want: struct {
				Foo string
				Bar map[string]string
			}{
				Foo: "don't modify",
				Bar: map[string]string{
					"don't": "modify",
				},
			},
		},
	}

	for _, c := range cases {
		t.Logf("Running: %s", c.name)

		// Test case consistency
		switch k := reflect.ValueOf(c.into).Kind(); k {
		// Reference types
		case reflect.Slice:
			fallthrough
		case reflect.Map:
			if got, want := reflect.ValueOf(c.want).Type(), reflect.ValueOf(c.into).Type(); got != want {
				t.Fatalf("reflect.ValueOf(c.want).Type(): got %v, want %v", got, want)
			}
		default:
			if got, want := reflect.ValueOf(c.into).Kind(), reflect.Ptr; got != want {
				t.Fatalf("reflect.ValueOf(c.into).Kind(): got %v, want %v", got, want)
			}
			if got, want := reflect.ValueOf(c.want).Type(), reflect.ValueOf(c.into).Elem().Type(); got != want {
				t.Fatalf("reflect.ValueOf(c.want).Type(): got %v, want %v", got, want)
			}
		}

		if err := form.Unmarshal(c.values, c.into, c.name); err != nil {
			t.Fatalf("form.Unmarshal error: %v", err)
		}

		switch k := reflect.ValueOf(c.into).Kind(); k {
		// Reference types
		case reflect.Slice:
			fallthrough
		case reflect.Map:
			if got, want := reflect.DeepEqual(c.into, c.want), true; got != want {
				t.Logf("\tc.into: %v", c.into)
				t.Logf("\tc.want: %v", c.want)
				t.Errorf("\treflect.DeepEqual(c.into, c.want): got: %t, want %t", got, want)
			}
		default:
			if got, want := reflect.DeepEqual(reflect.ValueOf(c.into).Elem().Interface(), c.want), true; got != want {
				t.Logf("\tc.into: %v", reflect.ValueOf(c.into).Elem().Interface())
				t.Logf("\tc.want: %v", c.want)
				t.Errorf("\treflect.DeepEqual(reflect.ValueOf(c.into).Elem().Interface(), c.want): got: %t, want %t", got, want)
			}
		}
	}
}
