package service

import "fmt"

// Error wraps an HTTP status code and message for handlers to translate into responses.
type Error struct {
	Code    int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// Unwrap exposes the inner error.
func (e *Error) Unwrap() error {
	return e.Err
}
