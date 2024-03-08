// http handler to serve filesystem files

package http

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// FileServer is a http handler to serve filesystem files
type FileServer struct {

	// the virtual directory will ber replaced with the localDir directory to found the local file
	virtualDir string // The virtual directory to serve
	localDir   string // The localDir directory to serve
	localDirFS fs.FS
	mux        *http.ServeMux
}

// ServeHTTP serves the http request implementing the http.Handler interface
func (s *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		s.mux.ServeHTTP(w, r)
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
	return path.Join(s.localDir, relativePath)
}

// Get the file from the localDir directory
func (s *FileServer) Get(w http.ResponseWriter, r *http.Request) {
	http.FileServerFS(s.localDirFS).ServeHTTP(w, r)
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

// NewFileServerHandler creates a new http handler to serve filesystem files
func NewFileServerHandler(pattern, localDir string) http.Handler {

	pattern = strings.TrimSuffix(path.Clean(pattern), "/") + "/"

	s := &FileServer{

		virtualDir: pattern,
		localDir:   localDir,
		localDirFS: os.DirFS(localDir),
		mux:        http.NewServeMux(),
	}

	s.mux.HandleFunc("GET "+s.virtualDir+"{$}", s.Get)
	s.mux.HandleFunc("POST "+s.virtualDir+"{$}", s.Post)
	s.mux.HandleFunc("PUT "+s.virtualDir+"{$}", s.Put)
	s.mux.HandleFunc("PATCH "+s.virtualDir+"{$}", s.Patch)
	s.mux.HandleFunc("DELETE "+s.virtualDir+"{$}", s.Delete)
	s.mux.HandleFunc("OPTIONS "+s.virtualDir+"{$}", s.Option)
	s.mux.HandleFunc("TRACE "+s.virtualDir+"{$}", s.Option)
	return s
}
