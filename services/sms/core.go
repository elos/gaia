package sms

import (
	"fmt"
	"net/http"
)

type Message struct {
	To, From PhoneNumber
	Body     string
}

type PhoneNumber string

func ExtractMessageFromRequest(r *http.Request) (*Message, error) {
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

	return &Message{
		From: PhoneNumber(from),
		To:   PhoneNumber(to),
		Body: body,
	}, nil
}
