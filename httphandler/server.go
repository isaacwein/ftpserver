package httphandler

import (
	"net/http"
	"time"
)

// Server is a wrapper around http.Server that provides additional functionality.
type Server struct {
	*http.Server
}

// TryListenAndServe starts the server and returns nil if the server started successfully within d, otherwise it returns an error.
func (s *Server) TryListenAndServe(d time.Duration) error {
	errC := make(chan error)
	go func() {
		err := s.Server.ListenAndServe()
		if err != nil {
			errC <- err
		}
	}()

	select {
	case err := <-errC:
		return err
	case <-time.After(d):
		return nil
	}
}

// TryListenAndServeTLS starts the server with the provided certFile and keyFile
// and returns nil if the server started successfully within d, otherwise it returns an error.
func (s *Server) TryListenAndServeTLS(certFile, keyFile string, d time.Duration) (err error) {
	errC := make(chan error)

	go func() {
		err = s.Server.ListenAndServeTLS(certFile, keyFile)
		if err != nil {
			errC <- err
		}
	}()

	select {
	case err = <-errC:
		return err
	case <-time.After(d):
		return nil
	}
}
