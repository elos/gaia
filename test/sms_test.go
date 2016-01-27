package test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"golang.org/x/net/context"
)

type mockSMSMSG struct {
	to, body string
}

type mockSMS struct {
	bus chan *mockSMSMSG
}

func (m *mockSMS) Send(to, body string) error {
	go func() {
		m.bus <- &mockSMSMSG{
			to:   to,
			body: body,
		}
	}()

	return nil
}

func newMockSMS() *mockSMS {
	return &mockSMS{
		bus: make(chan *mockSMSMSG),
	}
}

func TestCommandSMS(t *testing.T) {
	db := mem.NewDB()

	sms := newMockSMS()
	mux := services.NewSMSMux()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go mux.Start(ctx, db, sms)

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	g := gaia.New(
		ctx,
		&gaia.Middleware{},
		&gaia.Services{
			Logger:             services.NewTestLogger(t),
			DB:                 db,
			SMSCommandSessions: mux,
			WebCommandSessions: services.NewWebMux(),
		},
	)

	s := httptest.NewServer(g)
	defer s.Close()

	u, _ := testUser(t, db)

	t.Log("Creating profile")
	p := models.NewProfile()
	p.SetID(db.NewID())
	phone := "650 123 4567"
	p.Phone = phone
	p.SetOwner(u)
	if err := db.Save(p); err != nil {
		t.Fatal(err)
	}
	t.Log("Created")

	messageBody := "todo"

	params := url.Values{}
	params.Set("To", phone) // /command/ ignores this, twilio sends it though
	params.Set("From", phone)
	params.Set("Body", messageBody)
	url := s.URL + "/command/sms/?" + params.Encode()
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

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status code of %d", http.StatusNoContent)
	}

	select {
	case m := <-sms.bus:
		t.Logf("Message:\n%+v", m)

		if !strings.Contains(m.body, "elos") {
			t.Fatal("The message should have almost certainly contained the word elos")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for sms message")
	}
}

func TestCommandSMSInput(t *testing.T) {
	db := mem.NewDB()

	sms := newMockSMS()
	mux := services.NewSMSMux()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go mux.Start(ctx, db, sms)

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	g := gaia.New(
		ctx,
		&gaia.Middleware{},
		&gaia.Services{
			Logger:             services.NewTestLogger(t),
			DB:                 db,
			SMSCommandSessions: mux,
			WebCommandSessions: services.NewWebMux(),
		},
	)

	s := httptest.NewServer(g)
	defer s.Close()

	u, _ := testUser(t, db)

	t.Log("Creating profile")
	p := models.NewProfile()
	p.SetID(db.NewID())
	phone := "650 123 4567"
	p.Phone = phone
	p.SetOwner(u)
	if err := db.Save(p); err != nil {
		t.Fatal(err)
	}
	t.Log("Created")

	messageBody := "todo new"
	fakeSMS(t, s, phone, phone, messageBody)

	select {
	case m := <-sms.bus:
		t.Logf("Message:\n%+v", m)

		if !strings.Contains(m.body, "Name") {
			t.Fatal("The message should have asked for name of task")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for sms message")
	}

	taskName := "task name" // for the name
	fakeSMS(t, s, phone, phone, taskName)

	select {
	case m := <-sms.bus:
		t.Logf("Message:\n%+v", m)

		if !strings.Contains(m.body, "deadline") {
			t.Fatal("The message should have asked if it had a deadline")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for sms message")
	}

	fakeSMS(t, s, phone, phone, "n") // no deadline

	select {
	case m := <-sms.bus:
		t.Logf("Message:\n%+v", m)

		if !strings.Contains(m.body, "prerequisites") {
			t.Fatal("The message should have asked if it had a prereqs")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for sms message")
	}

	fakeSMS(t, s, phone, phone, "n") // no prereqs
	select {
	case m := <-sms.bus:
		t.Logf("Message:\n%+v", m)

		if !strings.Contains(m.body, "created") {
			t.Fatal("Should have indicated task was created")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for sms message")
	}

	task := models.NewTask()

	// ensure the task exists
	if err := db.PopulateByField("name", taskName, task); err != nil {
		t.Fatal(err)
	}

	t.Logf("Task:\n%+v", task)
}

func fakeSMS(t *testing.T, s *httptest.Server, from, to, body string) {
	params := url.Values{}
	params.Set("To", to) // /command/ ignores this, twilio sends it though
	params.Set("From", from)
	params.Set("Body", body)
	url := s.URL + "/command/sms/?" + params.Encode()
	t.Logf("Constructed URL: %s", url)

	fakePostSMS(t, url)
}

func fakePostSMS(t *testing.T, url string) {
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

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status code of %d", http.StatusNoContent)
	}
}
