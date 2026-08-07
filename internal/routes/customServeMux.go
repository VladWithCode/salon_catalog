package routes

import "net/http"

// customServeMux builds on top of http.ServeMux to provide the ability to customize
// the handler for not found routes
type customServeMux struct {
	*http.ServeMux

	// Must not try to WriteHeader as it is set to 404 by default
	//
	// Defaults to http.NotFoundHandler().ServeHTTP
	notFoundHandle http.HandlerFunc
}

func NewCustomServeMux() *customServeMux {
	return &customServeMux{
		http.NewServeMux(),
		http.NotFoundHandler().ServeHTTP,
	}
}

// Will search for the handler appropiate for the received request, if found
// processes the request normally, otherwise responds with a 404
func (csm *customServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, pattern := csm.Handler(r)

	if pattern == "" {
		csm.notFoundHandle(w, r)
		return
	}

	csm.ServeMux.ServeHTTP(w, r)
}

// Set the custom NotFoundHandler
func (csm *customServeMux) NotFoundHandleFunc(handler http.HandlerFunc) {
	csm.notFoundHandle = handler
}
