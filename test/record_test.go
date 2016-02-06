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
	"github.com/elos/gaia/routes"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

// --- Test Helpers (testInstance, testUser) {{{

func testInstance(t *testing.T, ctx context.Context) (data.DB, *gaia.Gaia, *httptest.Server) {
	db := mem.NewDB()

	g := gaia.New(
		ctx,
		&gaia.Middleware{},
		&gaia.Services{
			Logger:             services.NewTestLogger(t),
			DB:                 db,
			SMSCommandSessions: services.NewSMSMux(),
			WebCommandSessions: services.NewWebMux(),
		},
	)

	s := httptest.NewServer(g)

	return db, g, s
}

func testUser(t *testing.T, db data.DB) (*models.User, *models.Credential) {
	u, c, err := user.Create(db, "public", "private")
	if err != nil {
		t.Fatal(err)
	}

	return u, c
}

// --- }}}

// --- Test `GET /record/` {{{

func TestRecordGet(t *testing.T) {
	db, _, s := testInstance(t, context.Background())
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

// --- }}}

// --- Test `POST /record/` {{{

func TestRecordPost(t *testing.T) {
	db, _, s := testInstance(t, context.Background())
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

// --- }}}

// --- Test `DELETE /record/` {{{

func TestRecordDELETE(t *testing.T) {
	db, _, s := testInstance(t, context.Background())
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

// --- }}}

// --- Test `POST /record/query/` {{{

func TestRecordQuery(t *testing.T) {
	db, _, s := testInstance(t, context.Background())
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

func TestRecordQueryLimit(t *testing.T) {
	db, _, s := testInstance(t, context.Background())
	defer s.Close()

	user, cred := testUser(t, db)

	task1 := models.NewTask()
	task1.SetID(db.NewID())
	task1.CreatedAt = time.Now()
	task1.OwnerId = user.Id
	task1.Name = "task1"
	task1.UpdatedAt = time.Now()
	if err := db.Save(task1); err != nil {
		t.Fatal(err)
	}

	task2 := models.NewTask()
	task2.SetID(db.NewID())
	task2.CreatedAt = time.Now()
	task2.OwnerId = user.Id
	task2.Name = "task2"
	task2.UpdatedAt = time.Now()
	if err := db.Save(task2); err != nil {
		t.Fatal(err)
	}

	params := url.Values{}
	params.Set("kind", models.TaskKind.String())
	params.Set("limit", "1")
	url := s.URL + "/record/query/?" + params.Encode()
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

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code of %d", http.StatusOK)
	}

	// Should contain one or the other, but not both
	if strings.Contains(string(body), "task1") {
		if strings.Contains(string(body), "task2") {
			t.Fatal("Response body should have contained task2")
		}
	} else {
		if !strings.Contains(string(body), "task2") {
			t.Fatal("Response body should have contained task2")
		}
	}
}

func TestRecordQuerySkip(t *testing.T) {
	db, _, s := testInstance(t, context.Background())
	defer s.Close()

	user, cred := testUser(t, db)

	task1 := models.NewTask()
	task1.SetID(db.NewID())
	task1.CreatedAt = time.Now()
	task1.OwnerId = user.Id
	task1.Name = "task1"
	task1.UpdatedAt = time.Now()
	if err := db.Save(task1); err != nil {
		t.Fatal(err)
	}

	task2 := models.NewTask()
	task2.SetID(db.NewID())
	task2.CreatedAt = time.Now()
	task2.OwnerId = user.Id
	task2.Name = "task2"
	task2.UpdatedAt = time.Now()
	if err := db.Save(task2); err != nil {
		t.Fatal(err)
	}

	params := url.Values{}
	params.Set("kind", models.TaskKind.String())
	params.Set("skip", "1")
	url := s.URL + "/record/query/?" + params.Encode()
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

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code of %d", http.StatusOK)
	}

	// Should contain one or the other, but not both
	if strings.Contains(string(body), "task1") {
		if strings.Contains(string(body), "task2") {
			t.Fatal("Response body should have contained task2")
		}
	} else {
		if !strings.Contains(string(body), "task2") {
			t.Fatal("Response body should have contained task2")
		}
	}
}

// --- }}}

type taskChange struct {
	Record          models.Task `json:"record"`
	data.ChangeKind `json:"kind"`
}

func TestRecordChanges(t *testing.T) {
	ctx, cancelAllConnections := context.WithCancel(context.Background())
	defer cancelAllConnections()

	db, _, s := testInstance(t, ctx)
	defer s.Close()

	user, cred := testUser(t, db)

	serverURL := s.URL
	origin := serverURL
	wsURL := strings.Replace(serverURL, "http", "ws", 1)

	params := url.Values{}
	params.Set("public", cred.Public)
	params.Set("private", cred.Private)
	params.Set("kind", models.TaskKind.String())
	wsURL += routes.RecordChanges + "?" + params.Encode()
	t.Logf("Constructed URL: %s", wsURL)

	t.Log("Opening websocket")
	ws, err := websocket.Dial(wsURL, "", origin)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	t.Log("Websocket openened")

	time.Sleep(500 * time.Millisecond)

	t.Log("Creating task")
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
	t.Log("Task created")

	var tc taskChange
	if err := websocket.JSON.Receive(ws, &tc); err != nil {
		t.Fatal(err)
	}

	t.Logf("Task Change Recieved: %++v", tc)

	t.Log("Cancelling the connection")
	cancelAllConnections()
	t.Log("\tcancelled")

	if tc.ChangeKind != data.Update {
		t.Fatal("Expected ChangeKind to be Update")
	}

	if tc.Record.Name != taskName {
		t.Fatalf("Expected task name to be: '%s'", taskName)
	}
}
