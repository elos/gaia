package test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia"
	"github.com/elos/models"
)

func testInstance() (data.DB, *gaia.Gaia, *httptest.Server) {
	db := mem.NewDB()

	g := gaia.New(
		&gaia.Middleware{},
		&gaia.Services{
			DB: db,
		},
	)

	s := httptest.NewServer(g)

	return db, g, s
}

func testUser(t *testing.T, db data.DB) (*models.User, *models.Credential) {
	u := models.NewUser()
	u.SetID(db.NewID())
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	c := models.NewCredential()
	c.SetID(db.NewID())
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	c.Public = "public"
	c.Private = "private"
	c.OwnerId = u.Id

	if err := db.Save(u); err != nil {
		t.Fatal(err)
	}

	if err := db.Save(c); err != nil {
		t.Fatal(err)
	}

	return u, c
}

func TestRecordGet(t *testing.T) {
	db, _, s := testInstance()
	defer s.Close()

	user, cred := testUser(t, db)

	taskName := "task to retreive"
	task := models.NewTask()
	task.SetID(db.NewID())
	task.CreatedAt = time.Now()
	task.OwnerId = user.Id
	task.Name = taskName
	task.UpdatedAt = time.Now()
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	params := url.Values{}
	params.Set("kind", models.TaskKind.String())
	params.Set("id", task.ID().String())
	url := s.URL + "/record/?" + params.Encode()
	t.Logf("Constructed URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
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

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code of %d", http.StatusOK)
	}

	if !strings.Contains(string(body), taskName) {
		t.Fatal("Response body should have contained the task name")
	}
}

func TestRecordPost(t *testing.T) {
	db, _, s := testInstance()
	defer s.Close()

	user, cred := testUser(t, db)

	taskName := "task to modify"
	task := models.NewTask()
	task.SetID(db.NewID())
	task.CreatedAt = time.Now()
	task.OwnerId = user.Id
	task.Name = taskName
	task.UpdatedAt = time.Now()
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}
	modifiedName := "modifiedName"
	task.Name = modifiedName

	params := url.Values{}
	params.Set("kind", models.TaskKind.String())
	params.Set("id", task.ID().String())
	url := s.URL + "/record/?" + params.Encode()
	t.Logf("Constructed URL: %s", url)

	requestBody, err := json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
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

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code of %d", http.StatusOK)
	}

	if !strings.Contains(string(body), modifiedName) {
		t.Fatal("Response body should have contained the modified task name")
	}

	modifiedTask := models.NewTask()
	modifiedTask.SetID(task.ID())

	if err := db.PopulateByID(modifiedTask); err != nil {
		t.Fatal(err)
	}

	if modifiedTask.Name != modifiedName {
		t.Fatal("Task should have updated name")
	}
}

func TestRecordDELETE(t *testing.T) {
	db, _, s := testInstance()
	defer s.Close()

	user, cred := testUser(t, db)

	taskName := "task to modify"
	task := models.NewTask()
	task.SetID(db.NewID())
	task.CreatedAt = time.Now()
	task.OwnerId = user.Id
	task.Name = taskName
	task.UpdatedAt = time.Now()
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	params := url.Values{}
	params.Set("kind", models.TaskKind.String())
	params.Set("id", task.ID().String())
	url := s.URL + "/record/?" + params.Encode()
	t.Logf("Constructed URL: %s", url)

	req, err := http.NewRequest("DELETE", url, nil)
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

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status code of %d", http.StatusNoContent)
	}

	if string(body) != "" {
		t.Fatalf("Response body should be empty, but got: '%s'", string(body))
	}

	if err := db.PopulateByID(task); err != data.ErrNotFound {
		t.Fatal("The task should not be able to be found")
	}
}

func TestRecordQuery(t *testing.T) {
	db, _, s := testInstance()
	defer s.Close()

	user, cred := testUser(t, db)

	taskName := "task to retreive"
	task := models.NewTask()
	task.SetID(db.NewID())
	task.CreatedAt = time.Now()
	task.OwnerId = user.Id
	task.Name = taskName
	task.UpdatedAt = time.Now()
	if err := db.Save(task); err != nil {
		t.Fatal(err)
	}

	params := url.Values{}
	params.Set("kind", models.TaskKind.String())
	params.Set("id", task.ID().String())
	url := s.URL + "/record/query/?" + params.Encode()
	t.Logf("Constructed URL: %s", url)

	requestBody, err := json.Marshal(data.AttrMap{
		"name": taskName,
	})

	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
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

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code of %d", http.StatusOK)
	}

	if !strings.Contains(string(body), taskName) {
		t.Fatal("Response body should have contained the task name")
	}
}
