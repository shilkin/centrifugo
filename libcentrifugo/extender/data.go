package extender

import "encoding/json"

type data struct {
	Ring     int64             `json:"ring"`
	Count    int64             `json:"count"`
	Messages []message         `json:"ids"`
	Extended []json.RawMessage `json:"extended,omitempty"`
}

func (d *data) AddExtendedData(extended string) {
	d.Extended = append(d.Extended, json.RawMessage(extended))
}

type message struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}
