package ocdavadapters

import "net/http"

// Adapter is an interface for the ocdavadapters
type Adapter func(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request)
