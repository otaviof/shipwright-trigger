package webhooks

import (
	"errors"
	"net/http"
)

// Interface describe the signature expected for the instances handling WebHook requests, coming from
// Git service providers.
type Interface interface {
	// ExtractRequestPayload parse and extract the request details, service provider specific.
	ExtractRequestPayload(*http.Request) (*RequestPayload, error)

	// ExtractBuildSelector extract the search parameters to select the Build objects related to the
	// WebHook request payload.
	ExtractBuildSelector(*RequestPayload) (*BuildSelector, error)

	// ValidateSignature verifies the request payload against the informed secret token.
	ValidateSignature(*RequestPayload, []byte) error
}

var (
	// ErrUnknownEventType event can't be identified, should be part of the request header.
	ErrUnknownEventType = errors.New("event type is not known")

	// ErrUnknownEventType the event type extracted is not in use or expected.
	ErrUnsupportedEventType = errors.New("event type is not supported")

	// ErrParsingEvent unable to parse the request payload (body).
	ErrParsingEvent = errors.New("unable to parse event payload")

	// ErrIncompleteEvent the request payload is not complete, may be empty.
	ErrIncompleteEvent = errors.New("incomplete event")
)
