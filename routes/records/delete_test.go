package records_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func TestDeletePOST(t *testing.T) {
	db := mem.NewDB()

	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create error: %v", err)
	}

	e := &models.Event{
		Id:      db.NewID().String(),
		OwnerId: u.Id,
		Name:    "event name",
		Data: map[string]interface{}{
			"sensor": 45.3,
		},
	}

	if err := db.Save(e); err != nil {
		t.Fatalf("db.Save error: %v", err)
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "/records/delete/") {
			records.DeletePOST(user.NewContext(context.Background(), u), w, r, db, services.NewTestLogger(t))
		}

		records.QueryGET(user.NewContext(context.Background(), u), w, r, db, services.NewTestLogger(t))
	}))

	p := s.URL + "/records/delete/?" + url.Values{
		"kind": []string{"event"},
		"id":   []string{e.Id},
	}.Encode()

	resp, err := http.Post(p, "", new(bytes.Buffer))
	if err != nil {
		t.Fatalf("http.Post error: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll error: %v", err)
	}
	t.Logf("resp.Body:\n%s", body)

	if got, want := db.PopulateByID(e), data.ErrNotFound; got != want {
		t.Fatalf("db.PopulateByID: got %v, want %v", got, want)
	}
}
