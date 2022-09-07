package waybar

type Message struct {
	Class      []string `json:"class"`
	Text       string   `json:"text"`
	Percentage *uint    `json:"percentage"`
	Tooltip    string   `json:"tooltip"`
	Alt        string   `json:"alt"`
}
