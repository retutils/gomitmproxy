package proxy

// WebSocketMessage
type WebSocketMessage struct {
	Type       int
	Data       []byte
	FromClient bool // true if message is from client, false if from server
}
