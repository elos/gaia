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

// EditGET handles a `GET` request to the `/records/edit/` route of the records web UI.
//
// Parameters:
//		{
//			kind string
//			id string
//		}
//
// EditGET presents an HTML form containing the record under question.
//
// Success:
//		* StatusOK
//			- html page with form to edit record
// Errors:
//		* StatusBadRequest
//			- missing kind
//			- unrecognized kind
//			- missing id
//			- invalid id
//		* StatusUnauthorized
//			- access.CanWrite false
//		* StatusNotFound
//			- db.PopulateByID data.ErrNotFound
//		* StatusInternalServerError
//			- r.ParseForm error
//			- db.PopulateByID error
//			- ctx missing user
//			- access.CanWrite error
//			- form.Marshal error
//			- EditTemplate.Execute error
func EditGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
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

	i := r.FormValue("id")
	if i == "" {
		http.Error(w, "Missing parameter \"id\"", http.StatusBadRequest)
		return
	}

	id, err := db.ParseID(i)
	if err != nil {
		http.Error(w, "Invalid \"id\"", http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)
	m.SetID(id)
	switch err := db.PopulateByID(m); err {
	case data.ErrNotFound:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return

	case nil:
	}

	u, ok := user.FromContext(ctx)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ok, err = access.CanWrite(db, u, m)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	b, err := form.Marshal(m, k)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := EditTemplate.Execute(w, &EditData{
		FormHTML:   template.HTML(string(b)),
		SubmitText: "Update",
		Kind:       k,
		ID:         i,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// EditPOST handles a `POST` request to the `/records/edit/` routes of the records web UI.
//
// Parameters:
//		{
//			kind string
//			id string
//			<arbitrary-fields> {string|[]string}
//		}
//
// EditPOST updates the record by matching `kind` and `id` parameters, and dynamically pulls the approriate attributes
// from the form parameters.
//
// Success:
//		* StatusFound
//			- record updated, redirect to /records/view/
//
// Errors:
//		* StatusBadRequest
//			- missing kind
//			- unrecognized kind
//			- missing id
//			- invalid id
//		* StatusUnauthorized
//			- model not property
//			- access.CanWrite false
//		* StatusNotFound
//			- db.PopulateByID data.ErrNotFound
//		* StatusInternalServerError
//			- r.ParseForm error
//			- unmarshalling model
//			- db.PopulateByID error
//			- ctx missing user
//			- access.CanWrite error
//			- db.Save error
func EditPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
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

	i := r.FormValue("id")
	if i == "" {
		http.Error(w, "Missing parameter \"id\"", http.StatusBadRequest)
		return
	}

	id, err := db.ParseID(i)
	if err != nil {
		http.Error(w, "Invalid \"id\"", http.StatusBadRequest)
		return
	}

	m := models.ModelFor(kind)
	m.SetID(id)
	if err := db.PopulateByID(m); err != nil {
		if err == data.ErrNotFound {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := form.Unmarshal(r.Form, m, k); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
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

	ok, err = access.CanWrite(db, u, prop)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
