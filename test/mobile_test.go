package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/elos/data"
	"github.com/elos/gaia/routes"
	"github.com/elos/models"
	"golang.org/x/net/context"
)

func TestMobileLocationPOST(t *testing.T) {
	db, _, s := testInstance(t, context.Background())
	defer s.Close()

	_, cred := testUser(t, db)

	altitude, latitude, longitude := 50.0, 60.0, 70.0

	params := url.Values{}
	params.Set("altitude", strconv.FormatFloat(altitude, 'E', -1, 64))
	params.Set("latitude", strconv.FormatFloat(latitude, 'E', -1, 64))
	params.Set("longitude", strconv.FormatFloat(longitude, 'E', -1, 64))
	url := s.URL + routes.MobileLocation + "?" + params.Encode()
	t.Logf("Constructed URL: %s", url)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth(cred.Public, cred.Private)

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

	var respData map[string]interface{}
	if err := json.Unmarshal(body, &respData); err != nil {
		t.Fatal(err)
	}

	t.Logf("Response data:\n%+v", respData)

	iter, err := db.Query(models.EventKind).Select(data.AttrMap{
		"id": respData["id"].(string),
	}).Execute()
	if err != nil {
		t.Fatal(err)
	}

	e := models.NewEvent()
	if !iter.Next(e) {
		t.Fatal("There should be at least one event")
	}

	l, err := e.Location(db)
	if err != nil {
		t.Fatal(err)
	}

	if l.Altitude != altitude {
		t.Fatal("Altitudes don't match")
	}
}
