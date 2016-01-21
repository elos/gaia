package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/elos/data"
	"github.com/elos/models"
)

func TestRegisterPOST(t *testing.T) {
	db, _, s := testInstance(t)
	defer s.Close()

	username, password := "public", "private"

	params := url.Values{}
	params.Set("username", username)
	params.Set("password", password)
	url := s.URL + "/register/?" + params.Encode()
	t.Logf("Constructed URL: %s", url)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Code: %d", resp.StatusCode)
	t.Logf("Body:\n%s", body)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status code of %d", http.StatusCreated)
	}

	if !strings.Contains(string(body), "id") {
		t.Fatal("Response body should have contained user id")
	}

	t.Log("Unmarshalling response body into user struct")
	u := models.NewUser()
	if err := json.Unmarshal(body, u); err != nil {
		t.Fatal(err)
	}
	t.Log("unmarshalled")

	iter, err := db.Query(models.CredentialKind).Select(data.AttrMap{"owner_id": u.Id}).Execute()
	if err != nil {
		t.Fatal(err)
	}

	c := models.NewCredential()
	iter.Next(c)
	if err := iter.Close(); err != nil {
		t.Fatal(err)
	}

	if c.Public != username {
		t.Fatal("Credential's public should be the username")
	}

	if c.Private != password {
		t.Fatal("Credential's private should be the password")
	}
}
