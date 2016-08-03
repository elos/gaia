package routes_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"google.golang.org/grpc"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/x/auth"
	xdata "github.com/elos/x/data"
	xmodels "github.com/elos/x/models"
	"github.com/elos/x/records"
	"golang.org/x/net/context"
)

type mockWebUIClient struct {
	resp *records.HTTPResponse
	services.WebUIClient
}

func (m *mockWebUIClient) RegisterPOST(_ context.Context, _ *records.RegisterPOSTRequest, _ ...grpc.CallOption) (*records.HTTPResponse, error) {
	return m.resp, nil
}

func (m *mockWebUIClient) RegisterGET(_ context.Context, _ *records.RegisterGETRequest, _ ...grpc.CallOption) (*records.HTTPResponse, error) {
	return m.resp, nil
}

// TestRegisterPOST tests a POST request to the '/register/' endpoint
// in the happy case (i.e., the username and password are present).
// We verify:
//   * Response contains the user
//   * A user and a credential are created
//   * The user of the credential created is the user returned
func TestRegisterPOST(t *testing.T) {
	ctx := context.Background()
	db := mem.NewDB()

	dbclient, conn, err := xdata.DBBothLocal(db)
	if err != nil {
		t.Fatalf("xdata.DBBothLocal error: %v", err)
	}
	defer conn.Close()

	authclient, conn, err := auth.AuthBothLocal(dbclient)
	if err != nil {
		t.Fatalf("auth.AuthBothLocal error: %v", err)
	}
	defer conn.Close()

	webuiclient, conn, err := records.WebUIBothLocal(dbclient, authclient)
	if err != nil {
		t.Fatalf("records.WebUIBothLocal error: %v", err)
	}
	defer conn.Close()

	logger := services.NewTestLogger(t)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routes.RegisterPOST(ctx, w, r, db, logger, webuiclient)
	}))
	defer s.Close()

	resp, err := http.PostForm(s.URL, url.Values{
		"username": []string{"username"},
		"password": []string{"password"},
	})
	defer resp.Body.Close()

	if err != nil {
		t.Fatalf("error posting to RegisterPOST: %s", err)
	}

	t.Logf("Response:\n\t%v", resp)

	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Errorf("resp.Code: got %d, want %d", got, want)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading response body: %s", err)
	}

	u := new(models.User)
	if err := json.Unmarshal(bytes, u); err != nil {
		t.Fatalf("error unmarshalling user: %s", err)
	}

	t.Logf("User:\n\t%v", u)

	if len(u.Id) == 0 {
		t.Fatalf("len(u.Id) = 0, want non-empty id")
	}

	iter, err := db.Query(data.Kind(xmodels.Kind_USER.String())).Execute()
	if err != nil {
		t.Fatalf("db.Query(models.UserKind).Execute() error: %s", err)
	}

	users := mem.Slice(iter, func() data.Record { return new(models.User) })
	if got, want := len(users), 1; got != want {
		t.Fatalf("len(users): got %d, want %d", got, want)
	}

	iter, err = db.Query(data.Kind(xmodels.Kind_CREDENTIAL.String())).Execute()
	if err != nil {
		t.Fatal("db.Query(models.CredentialKind).Execute() error: %s", err)
	}

	creds := mem.Slice(iter, func() data.Record { return new(xmodels.Credential) })
	if got, want := len(creds), 1; got != want {
		t.Fatalf("len(creds): got %d, want %d", got, want)
	}

	t.Logf("Credential:\n%+v", creds[0])

	if got, want := creds[0].(*xmodels.Credential).OwnerId, u.Id; got != want {
		t.Fatalf("creds[0].OwnerId: got %s, want %s", got, want)
	}
}
