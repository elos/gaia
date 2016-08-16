package records

import "html/template"

const editTemplateRaw = `<html>
	<body>
		{{ with .Flash -}}
			{{ . }}
		{{- end }}

		<form method="post">
		{{ with .FormHTML }}
			{{ . }}
		{{ end }}
			{{ with .SubmitText }}
				<input type="submit" value="{{ . }}">
			{{ else }}
				<input type="submit" value="Save">
			{{ end }}
		</form>
		{{ if and .Kind .ID }}
		<a href="/records/view/?kind={{ .Kind }}&id={{ .ID }}"> View </a>
		{{ end }}
	</body>
</html>`

var EditTemplate = template.Must(
	template.New("records/edit").Parse(editTemplateRaw),
)

type EditData struct {
	Flash      string
	FormHTML   template.HTML
	SubmitText string

	Kind, ID string
}
