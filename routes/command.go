package routes

import (
	"io"
	"log"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/elos/command"
	"github.com/elos/gaia/services"
	"github.com/elos/gaia/services/sms"
	"github.com/elos/models/access"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

// Expects: From, To, Body params
func CommandSMSPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, logger services.Logger, sessions services.SMSCommandSessions) {
	m, err := sms.ExtractMessageFromRequest(r)

	if err != nil {
		logger.Fatal(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sessions.Inbound(m)

	w.WriteHeader(http.StatusNoContent)
}

func ContextualizeCommandWebGET(db data.DB, logger services.Logger) websocket.Handler {
	return func(c *websocket.Conn) {
		ctx := context.Background()

		if err := c.Request().ParseForm(); err != nil {
			logger.Print("Failure parsing form")
			return
		}

		public := c.Request().Form.Get("public")
		private := c.Request().Form.Get("private")

		if public == "" || private == "" {
			logger.Print("failed to retrieve credentials")
			return
		}

		cred, err := access.Authenticate(db, public, private)
		if err != nil {
			logger.Print("failed to auth")
			return
		}

		u, _ := cred.Owner(db)
		CommandWebGET(context.WithValue(ctx, userKey, u), c, logger, db)
	}
}

// Expects: the context to hold the authed user
func CommandWebGET(ctx context.Context, ws *websocket.Conn, logger services.Logger, db data.DB) {
	user, ok := userFromContext(ctx)
	if !ok {
		logger.Print("CommandWebGET Error: failed to retrieve user from context")
		return
	}

	input := make(chan string)
	output := make(chan string)

	session := command.NewSession(
		user, db, input, output,
		func() {
			close(output)
		},
	)

	go session.Start()

	go func() {
		for {
			var message string
			err := websocket.Message.Receive(ws, &message)
			log.Printf("Recieved: %s", message)

			if err != nil {
				if err != io.EOF {
					log.Printf("Read fail: %s", err)
				}

				close(output) // should this know about output
				return
			}

			input <- message
		}
	}()

	for o := range output {
		log.Printf("Sending: %s", o)
		err := websocket.Message.Send(ws, o)

		if err != nil {
			log.Printf("Write fail: %s", err)
			return
		}
	}
}
