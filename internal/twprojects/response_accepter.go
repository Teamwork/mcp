package twprojects

import (
	"errors"
	"net/http"

	"github.com/teamwork/twapi-go-sdk"
)

var _ twapi.HTTPResponser = (*responseAccepter[twapi.HTTPResponser])(nil)

// acceptedError is a sentinel error value that indicates a request was accepted
// but not immediately processed. This allows callers to check for this specific
// case and handle it separately from other errors that may occur when handling
// the HTTP response.
var acceptedError = errors.New("request accepted")

// responseAccepter is a wrapper around an HTTPResponser that checks for a 202
// Accepted status code and returns a specific error if it is encountered. This
// is used to handle cases where the Teamwork API accepts a request but does not
// immediately process it, allowing the caller to handle this case separately.
type responseAccepter[T twapi.HTTPResponser] struct {
	Responser T
}

// HandleHTTPResponse checks the HTTP response for a 202 Accepted status code
// and returns a specific error if it is encountered. Otherwise, it delegates to
// the underlying Handler to process the response as usual.
func (ra responseAccepter[T]) HandleHTTPResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusAccepted {
		return acceptedError
	}
	return ra.Responser.HandleHTTPResponse(resp)
}
