package services

import (
	"log"

	"github.com/subosito/twilio"
)

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
