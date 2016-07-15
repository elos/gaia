package records

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/elos/data"
	"github.com/elos/gaia/routes/records/form"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

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
	</body>
</html>`

var EditTemplate = template.Must(
	template.New("records/edit").Parse(editTemplateRaw),
)

type EditData struct {
	Flash      string
	FormHTML   template.HTML
	SubmitText string
}

// CreateGET handles a `GET` request to the `/records/create/` route of the records web UI.
//
// Parameters:
//		{
//			kind string
//		}
//
// CreateGET provides a web frame to create a new record of the provided `kind`.
//
// Success:
//		* StatusOK
//			- html page with form to create the record
//
// Errors:
//		* StatusBadRequest:
//			- missing kind
//			- unrecognized kind
//		* StatusUnauthorized
//			- model.(access.Property) false
//		* StatusInternalServerError
//			- error parsing form
//			- error marshalling model into form
//			- EditTemplate.Execute error
func CreateGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, l services.Logger) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.FormValue("kind")
	if k == "" {
		http.Error(w, "Missing parameter: \"kind\"", http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	if !models.Kinds[kind] {
		http.Error(w, fmt.Sprintf("Unrecognized kind: %q", k), http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)

	if _, ok := m.(access.Property); !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	b, err := form.Marshal(m, k)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := EditTemplate.Execute(w, &EditData{
		Flash:      "The record has not yet been created, you must save",
		FormHTML:   template.HTML(string(b)),
		SubmitText: "Create",
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// CreatePOST handles a `POST` request to the `/records/create/` route of the records web UI.
//
// Parameters:
//		{
//			kind string
//			<arbitrary-fields> {string|[]string}
//		}
//
// CreatePOST creates the record by matching the `kind`, and dynamically pulls the appropriate attributes
// from the form parameters.
//
// Success:
//		* StatusFound
//			- record created, redirect to /records/view/
//
// Errors:
//		* StatusBadRequest
//			- missing kind
//			- unrecognized kind
//		* StatusUnauthorized
//			- model not property
//			- access.CanCreate false
//		* StatusInternalServerError
//			- r.ParseForm error
//			- unmarshalling model
//			- ctx missing user
//			- access.CanCreate error
//			- db.Save error
func CreatePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, l services.Logger) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.FormValue("kind")
	if k == "" {
		http.Error(w, "Missing parameter \"kind\"", http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	if !models.Kinds[kind] {
		http.Error(w, fmt.Sprintf("Unrecognized kind: %q", k), http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)
	if err := form.Unmarshal(r.Form, m, kind.String()); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if m.ID().String() == "" {
		m.SetID(db.NewID())
	}

	u, ok := user.FromContext(ctx)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	prop, ok := m.(access.Property)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	ok, err := access.CanCreate(db, u, prop)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if err := db.Save(m); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w, r,
		"/records/view/?"+url.Values{
			"kind": []string{m.Kind().String()},
			"id":   []string{m.ID().String()},
		}.Encode(),
		http.StatusFound,
	)
}
