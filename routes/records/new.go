package records

import (
	"text/template"

	"github.com/elos/data"
	"github.com/elos/metis"
	"github.com/elos/models"
)

const newTemplateRaw = `
<html>
	<body>
		{{ with .Flash -}}
			{{ . }}
		{{- end }}
		<table>
		<tbody>
			{{ range .Models }}
			<tr>
			<td>
				<form id="{{ .Kind }}" method="get" action="/records/create/">
					<input type="text" name="kind" value="{{ .Kind }}" /> </td>
				</form>
			<td> <button type="submit" form="{{ .Kind }}"> New </button> </td>
			</tr>
			</form>
			{{ end }}
		</tbody>
		</table>
	</body>
</html>
`

var NewTemplate = template.Must(template.New("records/new").Parse(newTemplateRaw))

type NewData struct {
	Flash  string
	Models map[data.Kind]*metis.Model
}

var instantiableModels = make(map[data.Kind]*metis.Model)

func init() {
	for kind, model := range models.Metis {
		switch kind {
		case models.UserKind:
		default:
			instantiableModels[kind] = model
		}
	}
}
