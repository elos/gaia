package records_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func TestCreateGET(t *testing.T) {
	db := mem.NewDB()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records.CreateGET(context.Background(), w, r, db, services.NewTestLogger(t))
	}))

	p := s.URL + "?" + url.Values{"kind": []string{"event"}}.Encode()

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
}

func TestCreatePOST(t *testing.T) {
	db := mem.NewDB()
	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create error: %v", err)
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records.CreatePOST(user.NewContext(context.Background(), u), w, r, db, services.NewTestLogger(t))
	}))

	p := s.URL + "?" + url.Values{
		"kind":          []string{"event"},
		"event/OwnerId": []string{u.ID().String()},
		"event/Name":    []string{"event name"},
		"event/Data":    []string{`{"sensor": 45}`},
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

	iter, err := db.Query(models.EventKind).Execute()
	if err != nil {
		t.Fatalf("db.Query error: %v", err)
	}

	e := new(models.Event)
	iter.Next(e)
	if err := iter.Close(); err != nil {
		t.Fatalf("iter.Close error: %v", err)
	}

	if got, want := e.Name, "event name"; got != want {
		t.Errorf("e.Name: got %q, want %q", got, want)
	}

	if got, want := e.OwnerId, u.ID().String(); got != want {
		t.Errorf("e.OwnerId: got %q, want %q", got, want)
	}

	if sensor := e.Data["sensor"]; sensor == nil {
		t.Fatal("e.Data[\"sensor\"]: nil")
	} else if got, want := sensor.(float64), 45.0; got != want {
		t.Errorf("e.Data[\"sensor\"]: got %f, want %f", got, want)
	}
}
