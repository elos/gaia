package form

import (
	"bytes"
	"fmt"
)

type Form struct {
	AcceptCharset,
	Action,
	Autocomplete,
	Enctype,
	Method,
	Name,
	Novalidate,
	Target string

	Value interface{}
}

func (f *Form) MarshalForm(namespace string) ([]byte, error) {
	b := new(bytes.Buffer)

	b.WriteString("<form")
	if f.AcceptCharset != "" {
		fmt.Fprintf(b, ` accept-charset="%s"`, f.AcceptCharset)
	}
	if f.Action != "" {
		fmt.Fprintf(b, ` action="%s"`, f.Action)
	}
	if f.Autocomplete != "" {
		fmt.Fprintf(b, ` autocomplete="%s"`, f.Autocomplete)
	}
	if f.Autocomplete != "" {
		fmt.Fprintf(b, ` autocomplete="%s"`, f.Autocomplete)
	}
	if f.Enctype != "" {
		fmt.Fprintf(b, ` enctype="%s"`, f.Enctype)
	}
	if f.Method != "" {
		fmt.Fprintf(b, ` method="%s"`, f.Method)
	}
	if f.Name != "" {
		fmt.Fprintf(b, ` name="%s"`, f.Name)
	}
	if f.Novalidate != "" {
		fmt.Fprintf(b, ` novalidate="%s"`, f.Novalidate)
	}
	if f.Target != "" {
		fmt.Fprintf(b, ` target="%s"`, f.Target)
	}
	b.WriteString(">")

	bytes, err := Marshal(f.Value, f.Name)
	if err != nil {
		return nil, err
	}
	b.Write(bytes)
	b.WriteString("</form>")

	return b.Bytes(), nil
}
