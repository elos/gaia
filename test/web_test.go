package test

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia"
	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

func TestCommandWeb(t *testing.T) {
	db := mem.NewDB()

	sms := newMockSMS()
	smsMux := services.NewSMSMux()
	webMux := services.NewWebMux()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go smsMux.Start(ctx, db, sms)
	go webMux.Start(ctx, db)

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	g := gaia.New(
		ctx,
		&gaia.Middleware{},
		&gaia.Services{
			Logger:             services.NewTestLogger(t),
			DB:                 db,
			SMSCommandSessions: smsMux,
			WebCommandSessions: webMux,
		},
	)

	s := httptest.NewServer(g)
	defer s.Close()

	_, cred := testUser(t, db)

	serverURL := s.URL

	origin := serverURL
	wsURL := strings.Replace(serverURL, "http", "ws", 1)

	params := url.Values{}
	params.Set("public", cred.Public)
	params.Set("private", cred.Private)
	wsURL += "/command/web/?" + params.Encode()
	t.Logf("Constructed URL: %s", wsURL)

	t.Log("Opening websocket")
	ws, err := websocket.Dial(wsURL, "", origin)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	t.Log("Websocket openened")

	messageBody := "todo"

	// Sending a message
	t.Logf("Sending message: %s", messageBody)
	websocket.Message.Send(ws, messageBody)
	t.Logf("Sent")

	t.Log("Recieving a message")
	var received string
	websocket.Message.Receive(ws, &received)
	t.Logf("Recieved message: %s", received)

	t.Log("Verifying the response contained the word 'elos'")
	if !strings.Contains(received, "elos") {
		t.Fatal("The message should have almost certainly contained the word elos")
	}
	t.Log("Verified")
}
