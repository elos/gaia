package services

import (
	"log"

	"github.com/elos/data"
	"github.com/elos/elos/command"
	"github.com/elos/gaia/services/sms"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

type SMSCommandSessions interface {
	Inbound(m *sms.Message)
}

type smsMux struct {
	inbound  chan *sms.Message
	sessions map[sms.PhoneNumber]*commandSessionInfo
}

func (s *smsMux) Inbound(m *sms.Message) {
	s.inbound <- m
}

func NewSMSMux() *smsMux {
	return &smsMux{
		inbound:  make(chan *sms.Message),
		sessions: make(map[sms.PhoneNumber]*commandSessionInfo),
	}
}

func (mux *smsMux) Start(ctx context.Context, db data.DB, sender SMS) {
	timeouts := make(chan sms.PhoneNumber)

Run:
	for {
		select {
		case m := <-mux.inbound:
			sessionInfo, sessionExists := mux.sessions[m.From]

			// instantiate one
			if !sessionExists {
				sessionInput := make(chan string)
				sessionOutput := make(chan string)

				// We want to forward the strings on the output
				// channel and send them as SMS
				go func(out <-chan string, from sms.PhoneNumber, timeouts chan<- sms.PhoneNumber) {
					for o := range out {
						// use the SMS interface to send the message
						err := sender.Send(string(from), o)

						// timeout if error sending message
						if err != nil {
							log.Print("Read off bailing")
							close(sessionInput)
							timeouts <- from
							return
						}
					}
				}(sessionOutput, m.From, timeouts)

				u, err := user.ForPhone(db, string(m.From))
				if err != nil {
					u = nil
				}

				session := command.NewSession(
					u, db, sessionInput, sessionOutput,
					func() {
						timeouts <- m.From
					})
				go session.Start()

				sessionInfo = &commandSessionInfo{
					input:   sessionInput,
					session: session,
				}

				mux.sessions[m.From] = sessionInfo
			}

			// forward the message
			go func(input chan<- string, text string) {
				input <- text
			}(sessionInfo.input, m.Body)
		case number := <-timeouts:
			if _, exists := mux.sessions[number]; exists {
				delete(mux.sessions, number)
			}
		// the context has been cancelled
		case <-ctx.Done():
			break Run
		}
	}

	// close all inputs
	for _, sessionInfo := range mux.sessions {
		close(sessionInfo.input)
	}
}

type commandSessionInfo struct {
	input   chan<- string
	session *command.Session
}
