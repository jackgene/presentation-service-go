package chat

type Message struct {
	Sender    string `json:"s"`
	Recipient string `json:"r"`
	Text      string `json:"t"`
}

func (m Message) String() string {
	return m.Sender + " to " + m.Recipient + ": " + m.Text
}
