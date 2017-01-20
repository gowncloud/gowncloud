package ocdavadapters

import "net/http"

type responseHijacker struct {
	writer  http.ResponseWriter
	status  int
	headers http.Header
	body    []byte
}

func newResponseHijacker(w http.ResponseWriter) *responseHijacker {
	return &responseHijacker{
		writer:  w,
		headers: make(http.Header),
		body:    []byte{},
	}
}

// Header exposes the responseHijackers header map
func (rh *responseHijacker) Header() http.Header {
	return rh.headers
}

// Write writes to the body of the responsehijacker. Rather than sending the response
// directly to the client, it stores the response in a buffer so it can be modified
// by adapters at a later time.
func (rh *responseHijacker) Write(b []byte) (int, error) {
	rh.body = append(rh.body, b...)
	return len(b), nil
}

// WriteHeader writes the header to the responseHijacker. Rather than sending the
// response directly to the client, it stores the response so it can be modified by
// adapters at a later time
func (rh *responseHijacker) WriteHeader(status int) {
	rh.status = status
}
