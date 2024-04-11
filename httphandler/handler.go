// http-handler handler to serve filesystem files

package httphandler

import (
	"fmt"
	"github.com/telebroad/fileserver/filesystem"
	"github.com/telebroad/fileserver/tools"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Users is the interface to find a user by username and password and return it
type Users interface {
	// VerifyUser returns a user by JWT, if the user is not found it returns an error
	VerifyUser(request *http.Request) (any, error)
}

// FileServer is a httphandler handler to serve filesystem files
type FileServer struct {

	// the virtual directory will ber replaced with the localDir directory to found the local file
	virtualDir string // The virtual directory to serve
	localDirFS filesystem.NewFS
	mux        *http.ServeMux
	logger     *slog.Logger
	users      Users
}

func (s *FileServer) SetLogger(l *slog.Logger) {
	s.logger = l
}
func (s *FileServer) Logger() *slog.Logger {
	if s.logger == nil {
		s.logger = slog.Default()
	}
	return s.logger.With("module", "http-server-handler")
}

// ServeHTTP serves the httphandler request implementing the httphandler.Handler interface
func (s *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var protocol string
	if r.TLS == nil {
		protocol = "http://"
	} else {
		protocol = "https://"
	}
	if s.users != nil {
		_, err := s.users.VerifyUser(r)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized! "+err.Error(), http.StatusUnauthorized)
			return
		}
	}
	s.Logger().Debug("ServeHTTP", "method", r.Method, "url", protocol+r.Host+r.URL.String(), "remote", r.RemoteAddr, "user-agent", r.UserAgent())

	lw := tools.NewHttpResponseWriter(w, s.Logger())

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		s.mux.ServeHTTP(lw, r)
	case http.MethodOptions:
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Get the local path of the file
func (s *FileServer) localPath(urlPath string) string {
	// Trim the virtual directory and prepend the localDir directory
	relativePath := strings.TrimPrefix(urlPath, s.virtualDir)
	return path.Join(s.localDirFS.RootDir(), relativePath)
}

// Get the file from the localDir directory
func (s *FileServer) Get(w http.ResponseWriter, r *http.Request) {
	http.FileServerFS(s.localDirFS.GetFS()).ServeHTTP(w, r)
}

// Post the file to the localDir directory
func (s *FileServer) Post(w http.ResponseWriter, r *http.Request) {

	randFileName := fmt.Sprintf("%s", time.Now().Format("2006-01-02_15-06-07.00000000_MST"))
	filePathExt, err := mime.ExtensionsByType(r.Header.Get("Content-Type"))
	if err != nil || len(filePathExt) == 0 {
		http.Error(w, "Error reading Content-Type", http.StatusBadRequest)
		return
	}
	randFileName = randFileName + filePathExt[0]

	filename := s.localPath(filepath.Join(r.URL.Path, randFileName))

	newFile, err := os.Create(filename)
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()
	_, err = io.Copy(newFile, r.Body)
	if err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "File %s created\nto upload a file with a file name use PUT method", filename)
}

// Put the file to the localDir directory
func (s *FileServer) Put(w http.ResponseWriter, r *http.Request) {
	filename := s.localPath(r.URL.Path)
	newFile, err := os.Create(filename)
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()
	_, err = io.Copy(newFile, r.Body)
	if err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "File %s updated", filename)
}

// Patch the file to the localDir directory
func (s *FileServer) Patch(w http.ResponseWriter, r *http.Request) {
	filename := s.localPath(r.URL.Path)

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	_, err = io.Copy(f, r.Body)
	if err != nil {
		http.Error(w, "Error appending to file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File %s updated", filename)
}

// Delete the file from the localDir directory
func (s *FileServer) Delete(w http.ResponseWriter, r *http.Request) {
	filename := s.localPath(r.URL.Path)
	err := os.Remove(filename)
	if err != nil {
		http.Error(w, "Error deleting file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File %s deleted", filename)
}

func (s *FileServer) Option(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	w.WriteHeader(http.StatusOK)
}

// NewFileServerHandler creates a new httphandler handler to serve filesystem files
// The pattern is the virtual directory to serve it will be stripped from the URL in the handler
func NewFileServerHandler(pattern string, localDirFS filesystem.NewFS, users Users) http.Handler {

	pattern = strings.TrimSuffix(path.Clean(pattern), "/") + "/"

	s := &FileServer{

		virtualDir: pattern,
		localDirFS: localDirFS,
		mux:        http.NewServeMux(),
		users:      users,
	}

	s.mux.Handle("GET /{pathname...}", http.StripPrefix(s.virtualDir, http.HandlerFunc(s.Get)))
	s.mux.Handle("POST /{pathname...}", http.StripPrefix(s.virtualDir, http.HandlerFunc(s.Post)))
	s.mux.Handle("PUT /{pathname...}", http.StripPrefix(s.virtualDir, http.HandlerFunc(s.Put)))
	s.mux.Handle("PATCH /{pathname...}", http.StripPrefix(s.virtualDir, http.HandlerFunc(s.Patch)))
	s.mux.Handle("DELETE /{pathname...}", http.StripPrefix(s.virtualDir, http.HandlerFunc(s.Delete)))
	s.mux.Handle("OPTIONS /{pathname...}", http.StripPrefix(s.virtualDir, http.HandlerFunc(s.Option)))
	s.mux.Handle("TRACE /{pathname...}", http.StripPrefix(s.virtualDir, http.HandlerFunc(s.Option)))

	//return http.StripPrefix(s.virtualDir, http.FileServerFS(s.localDirFS.GetFS()))
	return s
}
