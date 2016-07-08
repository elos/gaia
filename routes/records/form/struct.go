package form

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

func (f *Form) FormMarshal(namespace string) ([]byte, error) {
	return nil, nil
}
