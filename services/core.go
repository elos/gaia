package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/elos/data"
	"github.com/elos/elos/command"
	"github.com/elos/models"
	"github.com/mitchellh/cli"
	"github.com/subosito/twilio"
	"golang.org/x/net/context"
)

// satisfied by *log.Logger
type Logger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

type DB interface {
	data.DB
}

type SMSCommandSessions interface {
	Inbound(m *SMSMessage)
}

type SMS interface {
	Send(to, body string) error
}

type twilioSMS struct {
	c    *twilio.Client
	from string
}

func SMSFromTwilio(c *twilio.Client, from string) SMS {
	return &twilioSMS{
		c:    c,
		from: from,
	}
}

func (t *twilioSMS) Send(to, body string) error {
	_, _, err := t.c.Messages.SendSMS(t.from, to, body)
	if err != nil {
		log.Printf("*twilioSMS.Send Error: %s", err)
	}
	return err
}

func NewLogger(out io.Writer) *log.Logger {
	return log.New(out, "", log.Ldate|log.Ltime|log.Lshortfile)
}

type phoneNumber string

type SMSMessage struct {
	To, From phoneNumber
	Body     string
}

type smsMux struct {
	inbound  chan *SMSMessage
	sessions map[phoneNumber]*commandSessionInfo
}

func (s *smsMux) Inbound(m *SMSMessage) {
	s.inbound <- m
}

func NewSMSMux() *smsMux {
	return &smsMux{
		inbound:  make(chan *SMSMessage),
		sessions: make(map[phoneNumber]*commandSessionInfo),
	}
}

func (mux *smsMux) Start(ctx context.Context, db data.DB, sms SMS) {
	timeouts := make(chan phoneNumber)

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
				go func(out <-chan string, from phoneNumber, timeouts chan<- phoneNumber) {
					for o := range out {
						log.Printf("should try to send: '%s'", o)
						// use the SMS interface to send the message
						err := sms.Send(string(from), o)

						// timeout if error sending message
						if err != nil {
							log.Print("Read off bailing")
							timeouts <- from
							return
						}
					}
				}(sessionOutput, m.From, timeouts)

				u, err := models.UserForPhone(db, string(m.From))
				if err != nil {
					u = nil
				}

				session := &commandSession{
					user:   u,
					input:  sessionInput,
					output: sessionOutput,
					db:     db,
					timeout: func() {
						timeouts <- m.From
					},
				}
				go session.start()

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
	session *commandSession
}

type commandSession struct {
	// the user interacting with the session
	user *models.User

	// the db to use to execute the commands
	db data.DB

	// channel to read string input on
	input <-chan string

	// channel to send string output on
	output chan<- string

	// function to call indicating timeout exit
	timeout func()
}

func (s *commandSession) start() {
	if s.user == nil {
		s.output <- "Looks like you don't have an account, sorry :("
		s.timeout()
	}

	for i := range s.input {
		// we block, so that the text ui can read in our absence
		s.run(strings.Split(i, " "))
	}
}

// assumes user is defined
func (s *commandSession) run(args []string) {
	// construct a new CLI with name and version
	c := cli.NewCLI("elos", "0.0.1")
	c.Args = args
	ui := NewTextUI(s.input, s.output)
	c.Commands = map[string]cli.CommandFactory{
		"todo": func() (cli.Command, error) {
			return &command.TodoCommand{
				UI:     ui,
				UserID: s.user.Id,
				DB:     s.db,
			}, nil
		},
	}

	_, err := c.Run()
	if err != nil {
		log.Printf("commandsession info error: %s", err)
	}
}

// A TextUI is used for making command line interfaces
// more suitable for a medium in which you can only communicate
// strings, i.e., text messaging
type TextUI struct {
	in  <-chan string
	out chan<- string
	uid string
}

// Constructs a new text ui
func NewTextUI(in <-chan string, out chan<- string) *TextUI {
	return &TextUI{
		in:  in,
		out: out,
	}
}

// send is the abstraction for sending out
func (u *TextUI) send(txt string) {
	u.out <- txt
}

// Ask asks the user for input using the given query. The response is
// returned as the given string, or an error.
func (u *TextUI) Ask(s string) (string, error) {
	u.send(s)
	select {
	case msg := <-u.in:
		return msg, nil
	case <-time.After(5 * time.Minute):
		u.out <- "timeout"
		return "", fmt.Errorf("TextUI Ask, timeout")
	}
}

// AskSecret asks the user for input using the given query, but does not echo
// the keystrokes to the terminal.
func (u *TextUI) AskSecret(s string) (string, error) {
	return u.Ask(s)
}

// Output is called for normal standard output.
func (u *TextUI) Output(s string) {
	u.send(s)
}

// Info is called for information related to the previous output.
// In general this may be the exact same as Output, but this gives
// UI implementors some flexibility with output formats.
func (u *TextUI) Info(s string) {
	u.send(s)
}

func (u *TextUI) Error(s string) {
	u.send(s)
}

func (u *TextUI) Warn(s string) {
	u.send(s)
}

func Extract(r *http.Request) (*SMSMessage, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	from := r.Form.Get("From")
	if from == "" {
		return nil, fmt.Errorf("missing from parameter")
	}

	to := r.Form.Get("To")
	if to == "" {
		return nil, fmt.Errorf("missing to parameter")
	}

	body := r.Form.Get("Body")
	if body == "" {
		return nil, fmt.Errorf("missing body parameter")
	}

	return &SMSMessage{
		From: phoneNumber(from),
		To:   phoneNumber(to),
		Body: body,
	}, nil
}
