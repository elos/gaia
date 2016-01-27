package services

import (
	"log"

	"github.com/elos/data"
	"github.com/elos/elos/command"
	"github.com/elos/models"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

type SocketSession struct {
	User *models.User
	*websocket.Conn
}

type WebCommandSessions interface {
	Inbound(w *SocketSession)
}

type webMux struct {
	inbound  chan *SocketSession
	sessions map[data.ID]*commandSessionInfo
}

func (w *webMux) Inbound(session *SocketSession) {
	w.inbound <- session
}

func NewWebMux() *webMux {
	return &webMux{
		inbound:  make(chan *SocketSession),
		sessions: make(map[data.ID]*commandSessionInfo),
	}
}

type message string

func (mux *webMux) Start(ctx context.Context, db data.DB) {
	timeouts := make(chan data.ID)

Run:
	for {
		select {
		case socket := <-mux.inbound:
			sessionInfo, sessionExists := mux.sessions[socket.User.ID()]

			if sessionExists {
				websocket.Message.Send(socket.Conn, "A user may only have one command session at once")
				if err := socket.Conn.Close(); err != nil {
					log.Printf("Error closing socket: %s", err)
				}
				return
			}

			// otherwise instantiate one

			sessionInput := make(chan string)
			sessionOutput := make(chan string)

			var bail = func() {
				timeouts <- socket.User.ID()
			}

			// We want to forward strings on output channel
			// to the websocket
			go func(out <-chan string, uid data.ID) {
				for o := range out {
					log.Printf("Forwarding: %s", o)
					err := websocket.Message.Send(socket.Conn, o)

					// timeout if error sending message
					if err != nil {
						log.Print("Read off bailing")
						bail()
						return
					}
					log.Print("Forwarded")
				}
			}(sessionOutput, socket.User.ID())

			// Foward input
			go func(in chan<- string) {
				for {
					var incoming string
					err := websocket.Message.Receive(socket.Conn, &incoming)

					if err != nil {
						log.Printf("Error reading from socket: %s", err)
						bail()
						return
					}

					in <- incoming
				}
			}(sessionInput)

			session := command.NewSession(
				socket.User, db, sessionInput, sessionOutput,
				func() {
					timeouts <- socket.User.ID()
				})
			go session.Start()

			sessionInfo = &commandSessionInfo{
				input:   sessionInput,
				session: session,
			}

			mux.sessions[socket.User.ID()] = sessionInfo
		case uid := <-timeouts:
			if session, exists := mux.sessions[uid]; exists {
				close(session.input)
				delete(mux.sessions, uid)
			}
		case <-ctx.Done():
			break Run
		}
	}

	// close all inputs
	for _, sessionInfo := range mux.sessions {
		close(sessionInfo.input)
	}
}
