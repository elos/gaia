package records

import "text/template"

const viewTemplateRaw = `
<html>
	<body>
		{{ with .Flash }} {{ . }} {{ end }}
		{{ if .Record }}
		{{ with .Kind }} This is a "{{ . }}" {{ end }}
		<table>
			<thead>
				<tr> <th> Attribute </th> <th> Value </th>
			</thead>
			<tbody>
				{{ range $attr, $val := .Record -}}
				<tr> <td> {{ $attr }} </td> <td> {{ $val }} </td>  </tr>
				{{- end }}
			</tbody>
		</table>
		{{ else }}
			No Model
		{{ end }}
		{{ if and .Kind .ID }}
			<a href="/records/edit/?kind={{ .Kind }}&id={{ .ID }}"> Edit </a>
		{{ end }}
	</body>
</html>`

var ViewTemplate = template.Must(template.New("records/view").Parse(viewTemplateRaw))

type ViewData struct {
	Flash, Kind, ID string
	Record          map[string]interface{}
}
