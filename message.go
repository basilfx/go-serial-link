package link

// MessageType indicates the type of the message.
type MessageType int

// The different message types.
const (
	MessageRequest = iota
	MessageResponse
	MessageNotification
)

// Message contains a parsed representation of a serial message.
type Message struct {
	Type    MessageType
	ID      uint32
	Command string
}
