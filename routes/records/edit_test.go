package records_test

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/routes/records/form"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func TestEditTemplate(t *testing.T) {
	b, err := form.Marshal(new(models.Task), models.TaskKind.String())
	if err != nil {
		t.Fatalf("form.Marshal(task, string(models.TaskKind)) error: %v", err)
	}

	if err := records.EditTemplate.Execute(
		ioutil.Discard,
		&records.EditData{
			Flash:    "flash message",
			FormHTML: template.HTML(string(b)),
		}); err != nil {
		t.Fatalf("records.EditTemplate.Execute error: %v", err)
	}
}

func TestEditGET(t *testing.T) {
	db := mem.NewDB()
	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create error: %v", err)
	}

	e := &models.Event{
		Id:      db.NewID().String(),
		OwnerId: u.Id,
		Name:    "old name",
		Data: map[string]interface{}{
			"sensor": 4,
		},
	}

	if err := db.Save(e); err != nil {
		t.Fatalf("db.Save error: %v", err)
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records.EditGET(user.NewContext(context.Background(), u), w, r, db, services.NewTestLogger(t))
	}))

	p := s.URL + "?" + url.Values{
		"kind": []string{"event"},
		"id":   []string{e.Id},
	}.Encode()

	resp, err := http.Get(p)
	if err != nil {
		t.Fatalf("http.Get(%q) error: %v", p, err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll error: %v", err)
	}
	t.Logf("resp.Body:\n%s", body)

	if got, want := bytes.Contains(body, []byte(`event`)), true; got != want {
		t.Fatalf("bytes.Contains(body, %q): got %t, want %t", "event", got, want)
	}

	if got, want := bytes.Contains(body, []byte(`old name`)), true; got != want {
		t.Fatalf("bytes.Contains(body, %q): got %t, want %t", "old name", got, want)
	}
}

func TestEditPOST(t *testing.T) {
	db := mem.NewDB()
	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create error: %v", err)
	}

	e := &models.Event{
		Id:      db.NewID().String(),
		OwnerId: u.Id,
		Name:    "old name",
		Data: map[string]interface{}{
			"sensor": 4,
		},
		Time: time.Now(),
	}

	if err := db.Save(e); err != nil {
		t.Fatalf("db.Save error: %v", err)
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "/records/edit/") {
			records.EditPOST(user.NewContext(context.Background(), u), w, r, db, services.NewTestLogger(t))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`default test handler`))
	}))

	p := s.URL + "/records/edit/?" + url.Values{
		"kind":       []string{"event"},
		"id":         []string{e.Id},
		"event/Name": []string{"new eman"},
		"event/Data": []string{`{"sensor": 4, "sensor2": 9}`},
	}.Encode()

	resp, err := http.Post(p, "", new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.Post error: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("iotuil.ReadAll error: %v", err)
	}
	t.Logf("resp.Body:\n%s", body)

	ue := &models.Event{Id: e.Id}
	if err := db.PopulateByID(ue); err != nil {
		t.Fatalf("db.PopulateByID error: %v", err)
	}

	if got, want := ue.Name, "new eman"; got != want {
		t.Errorf("ue.Name: got %q, want %q", got, want)
	}

	if got, want := ue.Data, map[string]interface{}{"sensor": 4.0, "sensor2": 9.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("ue.Data: got %v, want %v", got, want)
	}

	if got, want := ue.Time, e.Time; got != want {
		t.Errorf("ue.Time: got %v, want %v", got, want)
	}

	if got, want := ue.OwnerId, e.OwnerId; got != want {
		t.Errorf("ue.OwnerId: got %q, want %q", got, want)
	}
}
