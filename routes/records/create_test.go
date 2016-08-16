package records_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/elos/data"
	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/services"
	"github.com/elos/models/user"
	"github.com/elos/x/models"
	records_test "github.com/elos/x/records_test"
	"golang.org/x/net/context"
)

func TestCreateGET(t *testing.T) {
	adb, dbc, ac, closers, err := records_test.ClientsFromState(data.State{})
	defer func() {
		records_test.CloseAll(closers)
	}()
	wui := records.NewWebUI(adb, ac)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records.CreateGET(context.Background(), w, r, db, services.NewTestLogger(t), wui)
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
	adb, dbc, ac, closers, err := records_test.ClientsFromState(data.State{
		models.Kind_USER: []*data.Record{
			*data.Record{
				Kind: models.Kind_USER,
				User: &models.User{
					Id: "1",
				},
			},
		},
		models.Kind_CREDENTIAL: []*data.Record{
			&data.Record{
				Kind: models.Kind_CREDENTIAL,
				Credential: &models.Credential{
					Id:      "2",
					OwnerId: "1",
					Type:    models.Credential_PASSWORD,
					Public:  "username",
					Private: "password",
				},
			},
		},
	})
	defer func() {
		records_test.CloseAll(closers)
	}()
	wui := records.NewWebUI(adb, ac)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records.CreatePOST(user.NewContext(context.Background(), u), w, r, db, services.NewTestLogger(t), webuiclient)
	}))

	p := s.URL + "?" + url.Values{
		"kind":             []string{"EVENT"},
		"EVENT/OwnerId":    []string{"1"},
		"EVENT/Name":       []string{"event name"},
		"EVENT/Quantities": []string{`[{"name": "sensor", "value": 45}]`},
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

	if err := data.CompareState(dbc, data.State{
		models.Kind_USER: []*data.Record{
			*data.Record{
				Kind: models.Kind_USER,
				User: &models.User{
					Id: "1",
				},
			},
		},
		models.Kind_CREDENTIAL: []*data.Record{
			&data.Record{
				Kind: models.Kind_CREDENTIAL,
				Credential: &models.Credential{
					Id:      "2",
					OwnerId: "1",
					Type:    models.Credential_PASSWORD,
					Public:  "username",
					Private: "password",
				},
			},
		},
		models.Kind_EVENT: []*data.Record{
			&data.Record{
				Kind: models.Kind_EVENT,
				Event: &models.Event{
					Id:      "3",
					OwnerId: "1",
					Name:    "event name",
					Quantities: []*Quantity{
						Name:  "sensor",
						Value: 45,
					},
				},
			},
		},
	}); err != nil {
		t.Errorf("data.CompareState error: %v", err)
	}
}
