package routes_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

// TestEventPOST tests a POST request to the '/event/' endpoint.
// in the happy case (i.e., all parameters present).
// We verify:
//   * The event's name
//   * The event's data
//   * The event's tags
func TestEventPOST(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()
	logger := services.NewTestLogger(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(ctx, w, r, logger, db)
		if !ok {
			t.Error("routes.Authenticate failed")
		}
		routes.EventPOST(ctx, w, r, db, logger)
	}))
	defer s.Close()

	_, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatalf("user.Create(db, \"username\", \"password\") error: %s", err)
	}

	u := s.URL + "?" + url.Values{
		"tags": []string{"tag1", "tag2"},
	}.Encode()

	body := map[string]interface{}{
		"name": "event name",
		"time": time.Now(),
		"data": map[string]interface{}{
			"arbitrary": []string{"data"},
			"here":      1.0,
			"foo": map[string]interface{}{
				"bar": "there",
			},
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal("json.Marshal(body) error: %s", err)
	}

	req, err := http.NewRequest("POST", u, bytes.NewBuffer(b))
	req.SetBasicAuth("username", "password")

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do(req) error: %s", err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Fatalf("resp.StatusCode: got %d, want %d", got, want)
	}

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ioutil.ReadAll(resp.Body) error: %s", err)
	}

	e := new(models.Event)
	if err := json.Unmarshal(b, e); err != nil {
		t.Fatalf("json.Unmarshal(b, e) error: %s", err)
	}

	t.Logf("Event:\n\t%v", e)

	if got, want := e.Name, "event name"; got != want {
		t.Errorf("e.Name: got %q, want %q", got, want)
	}

	if got, want := e.Data["foo"].(map[string]interface{})["bar"], "there"; got != want {
		t.Errorf("e.Data[\"foo\"][\"bar\"]: got %q, want %q", got, want)
	}

	if got, want := len(e.TagsIds), 2; got != want {
		t.Fatalf("len(e.TagsIds): got %d, want %d", got, want)
	}

	tags, err := e.Tags(db)
	if err != nil {
		t.Fatalf("e.Tags(db) error: %s", err)
	}

	if got, want := len(tags), 2; got != want {
		t.Fatalf("len(tags): got %d, want %d", got, want)
	}

	if got, want := tags[0].Name, "tag1"; got != want {
		t.Errorf("tags[0].Name: got %q, want %q", got, want)
	}

	if got, want := tags[1].Name, "tag2"; got != want {
		t.Errorf("tags[1].Name: got %q, want %q", got, want)
	}
}
