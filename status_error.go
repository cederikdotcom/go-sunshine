package sunshine

import (
	"fmt"
	"net/http"
)

// StatusError is returned when Sunshine answers a request with a non-2xx
// status.
//
// It exists so a caller can tell two very different things apart. A status
// error means the HTTP server accepted the connection and replied, so
// Sunshine is alive. A transport error, such as a refused connection, a
// timeout or a closed socket, means it is not answering at all.
//
// A health check that treats both as "unreachable" will kill a live stream
// because the credentials were wrong, which is the opposite of what the
// operator wants.
//
// The message is unchanged from the untyped error this replaces, so callers
// that only print it keep their current output.
type StatusError struct {
	Code int
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Body)
}

// Unauthorized reports whether Sunshine rejected the credentials rather than
// the request. Both codes mean the same thing here: the API is up, and this
// client may not use it.
func (e *StatusError) Unauthorized() bool {
	return e.Code == http.StatusUnauthorized || e.Code == http.StatusForbidden
}
