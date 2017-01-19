package dav

import "net/http"

type ResponseHijacker struct {
	writer  http.ResponseWriter
	status  int
	headers http.Header
	body    []byte
}

func NewResponseHijacker(w http.ResponseWriter) *ResponseHijacker {
	return &ResponseHijacker{
		writer:  w,
		headers: make(http.Header),
		body:    []byte{},
	}
}

func (rh *ResponseHijacker) Header() http.Header {
	return rh.headers
}

func (rh *ResponseHijacker) Write(b []byte) (int, error) {
	rh.body = append(rh.body, b...)
	return len(b), nil
}

func (rh *ResponseHijacker) WriteHeader(status int) {
	rh.status = status
}
