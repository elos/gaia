package routes

import (
	"net/http"
	"text/template"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

var indexTemplateRaw = `
<html>
	<body>
		<h1> Elos </h1>
		{{if . }}
			Hello User {{print .ID}}
		{{else }}
			<a href="/login/">Login</a>
		{{ end}}
	</body>
</html>
`

var indexTemplate = template.Must(template.New("index").Parse(indexTemplateRaw))

func IndexGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("IndexGet: ")

	u, _ := user.FromContext(ctx)
	// u may be nil
	if err := indexTemplate.Execute(w, u); err != nil {
		l.Fatal(err)
	}
}
