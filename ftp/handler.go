package ftp

import "fmt"

type ServeMux struct {
	handlers map[Command]HandlerFunc
}

func NewServeMux() *ServeMux {
	return &ServeMux{
		handlers: make(map[Command]HandlerFunc),
	}
}

// HandleFunc registers the handler for the given method.
// if method is "", the handler is used for all requests that do not match any other method.
func (mux *ServeMux) HandleFunc(method Command, handler HandlerFunc) {
	mux.handlers[method] = handler
}

// ServeFTP dispatches the request to the handler whose pattern most closely matches the request Method.
func (mux *ServeMux) ServeFTP(w ResponseWriter, r *Request) {

	if _, ok := mux.handlers[r.Method]; ok {
		go mux.handlers[r.Method](w, r)
	} else if _, ok := mux.handlers[""]; ok {
		go mux.handlers[""](w, r)
	} else {
		w.WriteHeader(StatusSyntaxError)
		fmt.Fprintf(w, "Unknown command.\r\n")
		return
	}

}

var DefaultServeMux = NewServeMux()
