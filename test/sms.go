package test

import "github.com/elos/gaia/services"

type mockSMSSessions struct {
	messages []*services.SMSMessage
}

func (m *mockSMSSessions) Inbound(mess *services.SMSMessage) {
	m.messages = append(m.messages, mess)
}

func newMockSMSSessions() *mockSMSSessions {
	return &mockSMSSessions{
		messages: make([]*services.SMSMessage, 0),
	}
}
