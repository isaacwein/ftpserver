package httphandler

import (
	"net/http"
	"time"
)

type Server struct {
	*http.Server
}

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
