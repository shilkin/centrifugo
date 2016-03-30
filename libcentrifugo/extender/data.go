package extender

import "encoding/json"

type data struct {
	Ring     int64     `json:"ring"`
	Count    int64     `json:"count"`
	Messages []message `json:"ids"`
	// Extended []json.RawMessage `json:"extended,omitempty"`
}

type message struct {
	Status   string          `json:"status"`
	ID       string          `json:"id"`
	Extended json.RawMessage `json:"extended,omitempty"`
}

func (m *message) AddExtendedData(extended string) {
	m.Extended = json.RawMessage(extended)
}
