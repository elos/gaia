package form_test

import (
	"strings"
	"testing"
	"time"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes/records/form"
	"github.com/elos/models/user"
)

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
		// Bool
		{
			name:      "bool_true",
			structure: true,
			output:    `<label for="bool_true">bool_true</label><input name="bool_true" type="checkbox" checked/>`,
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
<label for="[]int">[]int</label><textarea name="[]int">[
	1,
	2,
	3
]</textarea>
			`),
		},
		{
			name:      "[]string",
			structure: []string{"foo", "bar", "tod"},
			output: strings.TrimSpace(`
<label for="[]string">[]string</label><textarea name="[]string">[
	"foo",
	"bar",
	"tod"
]</textarea>
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
				b bool
			}{
				b: true,
			},
			output: `<fieldset><legend>bool_field_true</legend><label for="bool_field_true/b">b</label><input name="bool_field_true/b" type="checkbox" checked/><br></fieldset>`,
		},
		{
			name: "bool_field_false",
			structure: struct {
				b bool
			}{
				b: false,
			},
			output: `<fieldset><legend>bool_field_false</legend><label for="bool_field_false/b">b</label><input name="bool_field_false/b" type="checkbox" /><br></fieldset>`,
		},
		{
			name: "integer_field",
			structure: struct {
				i int
			}{
				i: 45,
			},
			output: `<fieldset><legend>integer_field</legend><label for="integer_field/i">i</label><input name="integer_field/i" type="number" value="45" /><br></fieldset>`,
		},
		{
			name: "float_field",
			structure: struct {
				f float64
			}{
				f: 54.3,
			},
			output: `<fieldset><legend>float_field</legend><label for="float_field/f">f</label><input name="float_field/f" type="number" value="54.300000" /><br></fieldset>`,
		},
		{
			name: "string_field",
			structure: struct {
				s string
			}{
				s: "foo bar",
			},
			output: `<fieldset><legend>string_field</legend><label for="string_field/s">s</label><input name="string_field/s" type="text" value="foo bar" /><br></fieldset>`,
		},
		{
			name: "nested_structure",
			structure: struct {
				s struct {
					i int
				}
			}{
				s: struct{ i int }{
					i: 5,
				},
			},
			output: `<fieldset><legend>nested_structure</legend><fieldset><legend>s</legend><label for="nested_structure/s/i">i</label><input name="nested_structure/s/i" type="number" value="5" /><br></fieldset><br></fieldset>`,
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
					foo string
					bar int
				}{
					foo: "foo",
					bar: 8,
				},
				Name: "Structure",
			},
			output: `<form action="/action/" method="post" name="Structure"><fieldset><legend>Structure</legend><label for="Structure/foo">foo</label><input name="Structure/foo" type="text" value="foo" /><br><label for="Structure/bar">bar</label><input name="Structure/bar" type="number" value="8" /><br></fieldset></form>`,
		},
		{
			name: "nil_pointer",
			structure: struct {
				s *struct {
					val string
				}
			}{
				s: nil,
			},
			output: `<fieldset><legend>nil_pointer</legend></fieldset>`,
		},
		{
			name: "nil_interface",
			structure: struct {
				val interface{}
			}{
				val: nil,
			},
			output: `<fieldset><legend>nil_interface</legend></fieldset>`,
		},
	}

	for _, c := range cases {
		t.Logf("Running: %s", c.name)
		bytes, err := form.Marshal(c.structure, c.name)
		if err != nil {
			t.Fatalf("form.Marshal error: %v", err)
		}

		out := string(bytes)
		if got, want := out, c.output; got != want {
			t.Errorf("output: got,\n%s\nwant,\n%s", got, want)
		}
	}
}

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
