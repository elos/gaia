package records

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/metis"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const createTemplateRaw = `
<html>
	<body>
		{{with .Flash}}
			{{.}}
		{{end}}

		{{$model := .Model}}
		{{$json := .JSON}}
		<form enctype="application/json" method="post" action="/records/create/">
			<fieldset>
				<legend>Traits</legend>
				{{range $traitName, $trait := $model.Traits}}
					{{if eq $trait.Type 0 }}{{/* Boolean */}}
						<label> {{$trait.Name}} </label>
						<input type="checkbox" name="trait-{{$trait.Name}}" value="{{index $json $traitName}}">
					{{else if eq $trait.Type 1}}{{/* Integer */}}
						<label> {{$trait.Name}} </label>
						<input type="number" name="trait-{{$trait.Name}}" value="{{index $json $traitName}}">
					{{else if eq $trait.Type 2}}{{/* Float */}}
						<label> {{$trait.Name}} </label>
						<input type="number" name="trait-{{$trait.Name}}" value="{{index $json $traitName}}">
					{{else if eq $trait.Type 3}}{{/* String */}}
						<label> {{$trait.Name}} </label>
						<input type="text" name="trait-{{$trait.Name}}" value="{{index $json $traitName}}">
					{{else if eq $trait.Type 4}}{{/* DateTime */}}
						<label> {{$trait.Name}} </label>
						<input type="datetime-local" name="trait-{{$trait.Name}}" value="{{index $json $traitName}}">
					{{else if eq $trait.Type 5}}{{/* BooleanList */}}
						<fieldset id="{{$traitName}}FieldSet">
							<legend> {{$trait.Name}} </legend>
							{{$values := index $json $traitName}}
							<table>
							<tbody id="{{$traitName}}-tbody">
							{{range $i, $value := $values}}
							<tr id="{{$traitName}}-{{$i}}">
								<td>
									<input type="checkbox" name="trait-{{$trait.Name}}" value="{{$value}}">
								</td>
								<td>
									<button type="button"  onclick="remove{{castJS $traitName}}(id{{castJS $traitName}}(i))">x</button>
								</td>
							</tr>
							{{end}}
							</tbody>
							</table>
							<button id="{{$traitName}}Adder" onclick="add{{castJS $traitName}}()" type="button"> + {{$traitName}} </button>
							<script>
							{{with $values}}
								var num{{castJS $traitName}} = {{ len . }};
							{{else}}
								var num{{castJS $traitName}} = 0;
							{{end}}
							var {{castJS $traitName}}AdderButton = document.getElementById("{{$traitName}}Adder");

							function id{{castJS $traitName}}(i /*int*/) {
								return "{{$traitName}}-" + i;
							}

							function add{{castJS $traitName}}() {
								var input = document.createElement("input");
								input.name = "trait-{{$traitName}}";
								input.type = "checkbox";

								var x = document.createElement("button");
								x.type = "button";
								x.innerHTML = "x";
								x.setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}(" + num{{castJS $traitName}} + "))");

								var row = document.createElement('tr');
								row.id = id{{castJS $traitName}}(num{{castJS $traitName}});
								row.appendChild(document.createElement('td'))
								row.appendChild(document.createElement('td'))
								row.children[0].appendChild(input);
								row.children[1].appendChild(x);

								document.getElementById("{{$traitName}}-tbody").appendChild(row);

								num{{castJS $traitName}}++;
							}

							function remove{{castJS $traitName}}(remID /*string*/) {
								var rows = [];
								for (var i = 0; i < num{{castJS $traitName}}; i++) {
									var id = id{{castJS $traitName}}(i);
									if (id === remID) {
										document.getElementById(id).remove();
									} else {
										rows.push(document.getElementById(id));
									}
								}

								for (var i = 0; i < rows.length; i++) {
									rows[i].setAttribute("id", id{{castJS $traitName}}(i));
									rows[i].children[1].children[0].setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}("+i+"))");
								}

								num{{castJS $traitName}}--;
							}
							</script>
						</fieldset>
					{{else if eq $trait.Type 6}}{{/* IntegerList */}}
						<fieldset id="{{$traitName}}FieldSet">
							<legend> {{$trait.Name}} </legend>
							{{$values := index $json $traitName}}
							<table>
							<tbody id="{{$traitName}}-tbody">
							{{range $i, $value := $values}}
							<tr id="{{$traitName}}-{{$i}}">
								<td>
									<input type="number" name="trait-{{$trait.Name}}" value="{{$value}}">
								</td>
								<td>
									<button type="button"  onclick="remove{{castJS $traitName}}(id{{castJS $traitName}}(i))">x</button>
								</td>
							</tr>
							{{end}}
							</tbody>
							</table>
							<button id="{{$traitName}}Adder" onclick="add{{castJS $traitName}}()" type="button"> + {{$traitName}} </button>
							<script>
							{{with $values}}
								var num{{castJS $traitName}} = {{ len . }};
							{{else}}
								var num{{castJS $traitName}} = 0;
							{{end}}
							var {{castJS $traitName}}AdderButton = document.getElementById("{{$traitName}}Adder");

							function id{{castJS $traitName}}(i /*int*/) {
								return "{{$traitName}}-" + i;
							}

							function add{{castJS $traitName}}() {
								var input = document.createElement("input");
								input.name = "trait-{{$traitName}}";
								input.type = "number";

								var x = document.createElement("button");
								x.type = "button";
								x.innerHTML = "x";
								x.setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}(" + num{{castJS $traitName}} + "))");

								var row = document.createElement('tr');
								row.id = id{{castJS $traitName}}(num{{castJS $traitName}});
								row.appendChild(document.createElement('td'))
								row.appendChild(document.createElement('td'))
								row.children[0].appendChild(input);
								row.children[1].appendChild(x);

								document.getElementById("{{$traitName}}-tbody").appendChild(row);

								num{{castJS $traitName}}++;
							}

							function remove{{castJS $traitName}}(remID /*string*/) {
								var rows = [];
								for (var i = 0; i < num{{castJS $traitName}}; i++) {
									var id = id{{castJS $traitName}}(i);
									if (id === remID) {
										document.getElementById(id).remove();
									} else {
										rows.push(document.getElementById(id));
									}
								}

								for (var i = 0; i < rows.length; i++) {
									rows[i].setAttribute("id", id{{castJS $traitName}}(i));
									rows[i].children[1].children[0].setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}("+i+"))");
								}

								num{{castJS $traitName}}--;
							}
							</script>
						</fieldset>
					{{else if eq $trait.Type 7}}{{/* StringList */}}
						<fieldset id="{{$traitName}}FieldSet">
							<legend> {{$trait.Name}} </legend>
							{{$values := index $json $traitName}}
							<table>
							<tbody id="{{$traitName}}-tbody">
							{{range $i, $value := $values}}
							<tr id="{{$traitName}}-{{$i}}">
								<td>
									<input type="text" name="trait-{{$trait.Name}}" value="{{$value}}">
								</td>
								<td>
									<button type="button"  onclick="remove{{castJS $traitName}}(id{{castJS $traitName}}(i))">x</button>
								</td>
							</tr>
							{{end}}
							</tbody>
							</table>
							<button id="{{$traitName}}Adder" onclick="add{{castJS $traitName}}()" type="button"> + {{$traitName}} </button>
							<script>
							{{with $values}}
								var num{{castJS $traitName}} = {{ len . }};
							{{else}}
								var num{{castJS $traitName}} = 0;
							{{end}}
							var {{castJS $traitName}}AdderButton = document.getElementById("{{$traitName}}Adder");

							function id{{castJS $traitName}}(i /*int*/) {
								return "{{$traitName}}-" + i;
							}

							function add{{castJS $traitName}}() {
								var input = document.createElement("input");
								input.name = "trait-{{$traitName}}";
								input.type = "text";

								var x = document.createElement("button");
								x.type = "button";
								x.innerHTML = "x";
								x.setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}(" + num{{castJS $traitName}} + "))");

								var row = document.createElement('tr');
								row.id = id{{castJS $traitName}}(num{{castJS $traitName}});
								row.appendChild(document.createElement('td'))
								row.appendChild(document.createElement('td'))
								row.children[0].appendChild(input);
								row.children[1].appendChild(x);

								document.getElementById("{{$traitName}}-tbody").appendChild(row);

								num{{castJS $traitName}}++;
							}

							function remove{{castJS $traitName}}(remID /*string*/) {
								var rows = [];
								for (var i = 0; i < num{{castJS $traitName}}; i++) {
									var id = id{{castJS $traitName}}(i);
									if (id === remID) {
										document.getElementById(id).remove();
									} else {
										rows.push(document.getElementById(id));
									}
								}

								for (var i = 0; i < rows.length; i++) {
									rows[i].setAttribute("id", id{{castJS $traitName}}(i));
									rows[i].children[1].children[0].setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}("+i+"))");
								}

								num{{castJS $traitName}}--;
							}
							</script>
						</fieldset>
					{{else if eq $trait.Type 8}}{{/* DateTimeList */}}
						<fieldset id="{{$traitName}}FieldSet">
							<legend> {{$trait.Name}} </legend>
							{{$values := index $json $traitName}}
							<table>
							<tbody id="{{$traitName}}-tbody">
							{{range $i, $value := $values}}
							<tr id="{{$traitName}}-{{$i}}">
								<td>
									<input type="datetime-local" name="trait-{{$trait.Name}}" value="{{$value}}">
								</td>
								<td>
									<button type="button"  onclick="remove{{castJS $traitName}}(id{{castJS $traitName}}(i))">x</button>
								</td>
							</tr>
							{{end}}
							</tbody>
							</table>
							<button id="{{$traitName}}Adder" onclick="add{{castJS $traitName}}()" type="button"> + {{$traitName}} </button>
							<script>
							{{with $values}}
								var num{{castJS $traitName}} = {{ len . }};
							{{else}}
								var num{{castJS $traitName}} = 0;
							{{end}}
							var {{castJS $traitName}}AdderButton = document.getElementById("{{$traitName}}Adder");

							function id{{castJS $traitName}}(i /*int*/) {
								return "{{$traitName}}-" + i;
							}

							function add{{castJS $traitName}}() {
								var input = document.createElement("input");
								input.name = "trait-{{$traitName}}";
								input.type = "datetime-local";

								var x = document.createElement("button");
								x.type = "button";
								x.innerHTML = "x";
								x.setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}(" + num{{castJS $traitName}} + "))");

								var row = document.createElement('tr');
								row.id = id{{castJS $traitName}}(num{{castJS $traitName}});
								row.appendChild(document.createElement('td'))
								row.appendChild(document.createElement('td'))
								row.children[0].appendChild(input);
								row.children[1].appendChild(x);

								document.getElementById("{{$traitName}}-tbody").appendChild(row);

								num{{castJS $traitName}}++;
							}

							function remove{{castJS $traitName}}(remID /*string*/) {
								var rows = [];
								for (var i = 0; i < num{{castJS $traitName}}; i++) {
									var id = id{{castJS $traitName}}(i);
									if (id === remID) {
										document.getElementById(id).remove();
									} else {
										rows.push(document.getElementById(id));
									}
								}

								for (var i = 0; i < rows.length; i++) {
									rows[i].setAttribute("id", id{{castJS $traitName}}(i));
									rows[i].children[1].children[0].setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}("+i+"))");
								}

								num{{castJS $traitName}}--;
							}
							</script>
						</fieldset>
					{{else if eq $trait.Type 9}}{{/* ID */}}
						<label> {{$trait.Name}} </label>
						<input type="text" name="trait-{{$trait.Name}}" disabled>
						<input style="{display:none}" type="text" name="trait-{{$trait.Name}}">
					{{else if eq $trait.Type 10}}{{/* IDList */}}
						<fieldset id="{{$traitName}}FieldSet">
							<legend> {{$trait.Name}} </legend>
							{{$values := index $json $traitName}}
							<table>
							<tbody id="{{$traitName}}-tbody">
							{{range $i, $value := $values}}
							<tr id="{{$traitName}}-{{$i}}">
								<td>
									<input type="text" name="trait-{{$trait.Name}}" value="{{$value}}">
								</td>
								<td>
									<button type="button"  onclick="remove{{castJS $traitName}}(id{{castJS $traitName}}(i))">x</button>
								</td>
							</tr>
							{{end}}
							</tbody>
							</table>
							<button id="{{$traitName}}Adder" onclick="add{{castJS $traitName}}()" type="button"> + {{$traitName}} </button>
							<script>
							{{with $values}}
								var num{{castJS $traitName}} = {{ len . }};
							{{else}}
								var num{{castJS $traitName}} = 0;
							{{end}}
							var {{castJS $traitName}}AdderButton = document.getElementById("{{$traitName}}Adder");

							function id{{castJS $traitName}}(i /*int*/) {
								return "{{$traitName}}-" + i;
							}

							function add{{castJS $traitName}}() {
								var input = document.createElement("input");
								input.name = "trait-{{$traitName}}";
								input.type = "text";

								var x = document.createElement("button");
								x.type = "button";
								x.innerHTML = "x";
								x.setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}(" + num{{castJS $traitName}} + "))");

								var row = document.createElement('tr');
								row.id = id{{castJS $traitName}}(num{{castJS $traitName}});
								row.appendChild(document.createElement('td'))
								row.appendChild(document.createElement('td'))
								row.children[0].appendChild(input);
								row.children[1].appendChild(x);

								document.getElementById("{{$traitName}}-tbody").appendChild(row);

								num{{castJS $traitName}}++;
							}

							function remove{{castJS $traitName}}(remID /*string*/) {
								var rows = [];
								for (var i = 0; i < num{{castJS $traitName}}; i++) {
									var id = id{{castJS $traitName}}(i);
									if (id === remID) {
										document.getElementById(id).remove();
									} else {
										rows.push(document.getElementById(id));
									}
								}

								for (var i = 0; i < rows.length; i++) {
									rows[i].setAttribute("id", id{{castJS $traitName}}(i));
									rows[i].children[1].children[0].setAttribute("onclick", "remove{{castJS $traitName}}(id{{castJS $traitName}}("+i+"))");
								}

								num{{castJS $traitName}}--;
							}
							</script>
						</fieldset>
					{{else if eq $trait.Type 13}}{{/* JSON */}}
						<label> {{$trait.Name}} </label>
						<input type="text" name="trait-{{$trait.Name}}">
					{{else}}
						Unrecognized primitive
					{{end}}
					<br />
				{{else}}
					No traits for this model.
				{{end}}
			</fieldset>
			<fieldset>
				<legend>Relations</legend>
				{{range $relationName, $relation := $model.Relations}}
					{{if eq $relation.Multiplicity 0}}{{/* Mul */}}
						<fieldset id="{{$relationName}}FieldSet">
							<legend> {{$relation.Name}} </legend>
							{{$values := index $json $relationName}}
							<table>
							<tbody id="{{$relationName}}-tbody">
							{{range $i, $value := $values}}
							<tr id="{{$relationName}}-{{$i}}">
								<td>
									<input type="text" name="relation-{{$relation.Name}}-id" value="{{$value}}">
								</td>
								<td>
									<button type="button"  onclick="remove{{castJS $relationName}}(id{{castJS $relationName}}(i))">x</button>
								</td>
							</tr>
							{{end}}
							</tbody>
							</table>
							<button id="{{$relationName}}Adder" onclick="add{{castJS $relationName}}()" type="button"> + {{$relationName}} </button>
							<script>
							{{with $values}}
								var num{{castJS $relationName}} = {{ len . }};
							{{else}}
								var num{{castJS $relationName}} = 0;
							{{end}}
							var {{castJS $relationName}}AdderButton = document.getElementById("{{$relationName}}Adder");

							function id{{castJS $relationName}}(i /*int*/) {
								return "{{$relationName}}-" + i;
							}

							function add{{castJS $relationName}}() {
								var input = document.createElement("input");
								input.name = "relation-{{$relationName}}-id";
								input.type = "text";

								var x = document.createElement("button");
								x.type = "button";
								x.innerHTML = "x";
								x.setAttribute("onclick", "remove{{castJS $relationName}}(id{{castJS $relationName}}(" + num{{castJS $relationName}} + "))");

								var row = document.createElement('tr');
								row.id = id{{castJS $relationName}}(num{{castJS $relationName}});
								row.appendChild(document.createElement('td'))
								row.appendChild(document.createElement('td'))
								row.children[0].appendChild(input);
								row.children[1].appendChild(x);

								document.getElementById("{{$relationName}}-tbody").appendChild(row);

								num{{castJS $relationName}}++;
							}

							function remove{{castJS $relationName}}(remID /*string*/) {
								var rows = [];
								for (var i = 0; i < num{{castJS $relationName}}; i++) {
									var id = id{{castJS $relationName}}(i);
									if (id === remID) {
										document.getElementById(id).remove();
									} else {
										rows.push(document.getElementById(id));
									}
								}

								for (var i = 0; i < rows.length; i++) {
									rows[i].setAttribute("id", id{{castJS $relationName}}(i));
									rows[i].children[1].children[0].setAttribute("onclick", "remove{{castJS $relationName}}(id{{castJS $relationName}}("+i+"))");
								}

								num{{castJS $relationName}}--;
							}
							</script>
						</fieldset>
					{{else if eq $relation.Multiplicity 1}}{{/* One */}}
						<label> {{$relation.Name}} </label>
						<input type="text" name="relation-{{$relation.Name}}-id" value="{{printf "%s%s" $relationName "_id" | index $json }}">
					{{end}}
				{{else}}
					No relations.
				{{end}}
			</fieldset>
			<input type="submit" value="Save" />
		</form>
	</body>
</html>
`

var CreateTemplate = template.Must(
	template.New("records/create").Funcs(template.FuncMap{
		"castJS": func(s string) template.JS {
			return template.JS(s)
		},
	}).Parse(createTemplateRaw),
)

type CreateData struct {
	Flash string
	Model *metis.Model
	JSON  map[string]interface{}
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
//		* StatusOK, return web frame
//
// Errors:
//		* StatusBadRequest, {no kind parameter found, kind not recognized}
//		* StatusInternalServerError, {error parsing, et al.}
func CreateGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, l services.Logger) {
	if err := r.ParseForm(); err != nil {
		l.Printf("r.ParseForm() error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("r.FormValue(kindParam): got %q, want a non-empty string", k)
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", kindParam), http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Lookup the kind to ensure its existence.
	if _, ok := models.Kinds[kind]; !ok {
		l.Printf("_, ok := models.Kind[kind]: got %t, want true", ok)
		http.Error(w, fmt.Sprintf("The kind %q is not recognized", kind), http.StatusNotFound)
		return
	}

	ed := &CreateData{
		Flash: "The record has not yet been created, you must save",
		Model: models.Metis[kind],
		JSON:  make(map[string]interface{}),
	}

	if err := CreateTemplate.Execute(w, ed); err != nil {
		l.Fatalf("CreateTemplate.Execute error: %v", err)
	}
}

// CreatePOST handles a `POST` request to the `/records/create/` route of the records web UI.
//
// Parameters:
//	{
//		kind string
//		<arbitrary-fields> {string|[]string}
//	}
//
// CreatePOST creates the record matching the `kind`, and dynamically pulls the appropriate attributes
// from the form parameters.
//
// Success:
//		* StatusFound, redirect to `/records/view/?kind=<dynamic>&id=<dynamic>` (i.e., redirects to viewing the record
//		which was just created
//
// Errors:
//		* 400, malformed parameters
//		* 404, kind parameter not recognized
//		* 500, {error parsing, et al.}
func CreatePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, l services.Logger) {
	if err := r.ParseForm(); err != nil {
		l.Printf("r.ParseForm() error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	k := r.FormValue(kindParam)
	if k == "" {
		l.Printf("r.FormValue(kindParam): got %q, want a non-empty string", k)
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", kindParam), http.StatusBadRequest)
		return
	}
	kind := data.Kind(k)

	// Lookup the kind to ensure its existence.
	if _, ok := models.Kinds[kind]; !ok {
		l.Printf("_, ok := models.Kind[kind]: got %t, want true", ok)
		http.Error(w, fmt.Sprintf("The kind %q is not recognized", kind), http.StatusNotFound)
		return
	}

	m := models.ModelFor(kind)

	var requestBody []byte
	var err error

	// Now we must read the body of the request
	defer r.Body.Close() // don't forget to close it
	if requestBody, err = ioutil.ReadAll(r.Body); err != nil {
		l.Printf("error while reading request body: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Now we unmarshal that into the record
	if err = json.Unmarshal(requestBody, m); err != nil {
		l.Printf("info: request body:\n%s", string(requestBody))
		l.Printf("error: while unmarshalling request body, %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	m.SetID(db.NewID())

	// Retrieve our user
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	prop, ok := m.(access.Property)
	if !ok {
		l.Printf("tried to create record that isn't property")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	allowed, err := access.CanCreate(db, u, prop)

	if err != nil {
		l.Printf("access.CanCreate error: %s", err)
		switch err {
		// This indicates that no, you have no access
		case data.ErrAccessDenial:
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		// All of these are bad, and considered an internal error
		case data.ErrNotFound:
			fallthrough
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
			fallthrough
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	} else if !allowed {
		l.Printf("access denied at create/update stage")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// If we have made it this far, it only remains to commit the record
	if err = db.Save(m); err != nil {
		l.Printf("error saving record: %s", err)
		switch err {
		case data.ErrAccessDenial:
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		// These are all equally distressing
		case data.ErrNotFound: // TODO shouldn't a not found not be fing impossible for a Save?
			fallthrough
		case data.ErrNoConnection:
			fallthrough
		case data.ErrInvalidID:
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/records/view/?kind=%s&id=%s", m.Kind(), m.ID()), http.StatusFound)
}
