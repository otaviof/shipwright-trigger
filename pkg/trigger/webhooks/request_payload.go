package webhooks

// RequestPayload the context of a webhook request.
type RequestPayload struct {
	EventType string // name of the event
	Signature string // request signature
	Payload   []byte // request payload
}
