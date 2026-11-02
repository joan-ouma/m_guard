package domain

import "time"

// ChatRequest carries the rider's menu selections so far, e.g. ["lang:sw", "section:bikes", "bike:<id>", "action:location"].
// The server is stateless: the path is the whole navigation state.
// Label is the text of the option the rider just tapped; it is stored with the answer in history.
type ChatRequest struct {
	Path  []string `json:"path"`
	Label string   `json:"label,omitempty"`
}

// ChatOption is one button the rider can tap. ID is sent back in the next request's path.
type ChatOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ChatResponse is either a menu ("menu", with Options) or a final answer ("answer", with no options).
type ChatResponse struct {
	Type    string       `json:"type"`
	Message string       `json:"message"`
	Options []ChatOption `json:"options"`
}

// ChatHistoryItem is one stored message. Role is "rider" (the tapped option) or "bot" (the answer).
type ChatHistoryItem struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
