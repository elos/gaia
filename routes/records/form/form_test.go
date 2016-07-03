package form_test

import (
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
			output:    `<input name="" type="datetime-local" value="2006-01-02T07:04:05-08:00" />`,
		},

		// Primitives
		// Bool
		{
			name:      "true bool",
			structure: true,
			output:    `<input name="" type="checkbox" checked/>`,
		},
		{
			name:      "false bool",
			structure: false,
			output:    `<input name="" type="checkbox" />`,
		},
		// Int
		{
			name:      "int",
			structure: int(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		{
			name:      "int8",
			structure: int8(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		{
			name:      "int16",
			structure: int16(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		{
			name:      "int32",
			structure: int32(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		{
			name:      "int64",
			structure: int64(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		// Uint
		{
			name:      "uint",
			structure: uint(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		{
			name:      "uint8",
			structure: uint8(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		{
			name:      "uint16",
			structure: uint16(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		{
			name:      "uint32",
			structure: uint32(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		{
			name:      "uint64",
			structure: uint64(5),
			output:    `<input name="" type="number" value="5" />`,
		},
		// Floats
		{
			name:      "float32",
			structure: float32(123.0),
			output:    `<input name="" type="number" value="123.000000" />`,
		},
		{
			name:      "float64",
			structure: float32(123.0),
			output:    `<input name="" type="number" value="123.000000" />`,
		},
		// String
		{
			name:      "string",
			structure: "here is a string",
			output:    `<input name="" type="text" value="here is a string" />`,
		},
		// Interface
		{
			name:      "interface of type string",
			structure: interface{}("string"),
			output:    `<input name="" type="text" value="string" />`,
		},

		// Composites
		// Slices
		{
			name:      "[]int",
			structure: []int{1, 2, 3},
			output:    "",
		},

		// Structures
		{
			name: "bool field",
			structure: struct {
				b bool
			}{
				b: true,
			},
			output: `<input name="b" type="checkbox" checked/>`,
		},
		{
			name: "bool field",
			structure: struct {
				b bool
			}{
				b: false,
			},
			output: `<input name="b" type="checkbox" />`,
		},
		{
			name: "integer field",
			structure: struct {
				i int
			}{
				i: 45,
			},
			output: `<input name="i" type="number" value="45" />`,
		},
		{
			name: "float field",
			structure: struct {
				f float64
			}{
				f: 54.3,
			},
			output: `<input name="f" type="number" value="54.300000" />`,
		},
		{
			name: "string field",
			structure: struct {
				s string
			}{
				s: "foo bar",
			},
			output: `<input name="s" type="text" value="foo bar" />`,
		},
		{
			name: "nested structure",
			structure: struct {
				s struct {
					i int
				}
			}{
				s: struct{ i int }{
					i: 5,
				},
			},
			output: `<input name="s[i]" type="number" value="5" />`,
		},
	}

	for _, c := range cases {
		t.Logf("Running: %s", c.name)
		bytes, err := form.Marshal(c.structure)
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

	bytes, err := form.Marshal(user)
	if err != nil {
		t.Fatalf("form.Marshal error: %v", err)
	}

	if got, want := string(bytes), ""; got != want {
		t.Fatalf("form.Marshal: got\n%s\nwant,\n%s", got, want)
	}
}
