package waybar

import (
	"encoding/json"
	"fmt"
	"os"
)

type Message struct {
	Class      []string `json:"class"`
	Text       string   `json:"text"`
	Percentage *uint    `json:"percentage"`
	Tooltip    string   `json:"tooltip"`
	Alt        string   `json:"alt"`
}

func (m Message) Emit() error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal json: %v", err)
	}
	data = append(data, '\n')
	_, err = os.Stdout.Write(data)
	return err
}
