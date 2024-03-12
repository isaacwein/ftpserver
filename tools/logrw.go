package tools

import (
	"bufio"
	"io"
	"log/slog"
	"net/http"
)

type LogReader struct {
	Reader io.Reader
	logger *slog.Logger
}

func (rw *LogReader) Read(b []byte) (int, error) {
	n, err := rw.Reader.Read(b)
	if rw.logger != nil && n > 0 { // Log only if n > 0 to avoid logging empty reads
		rw.logger.Debug("Request", "body", string(b[:n])) // Adjusted to log only the read portion
	}
	return n, err
}
func NewLogReader(r io.Reader, logger *slog.Logger) *LogReader {
	return &LogReader{Reader: r, logger: logger}
}

type LogWriter struct {
	Writer io.Writer
	logger *slog.Logger
}

func (rw *LogWriter) Write(b []byte) (int, error) {
	if rw.logger != nil {
		rw.logger.Debug("Respond", "body", string(b))
	}
	return rw.Writer.Write(b)
}
func NewLogWriter(w io.Writer, logger *slog.Logger) *LogWriter {
	return &LogWriter{Writer: w, logger: logger}

}

// LogReadWriter is a wrapper around a bufio.ReadWriter that logs all reads and writes to a slog.Logger.
type LogReadWriter struct {
	ReadWriter io.ReadWriter
	logger     *slog.Logger
}

func (rw *LogReadWriter) Read(b []byte) (int, error) {
	n, err := rw.ReadWriter.Read(b)
	if rw.logger != nil && n > 0 { // Log only if n > 0 to avoid logging empty reads
		rw.logger.Debug("Request", "body", string(b[:n])) // Adjusted to log only the read portion
	}
	return n, err
}
func (rw *LogReadWriter) Write(b []byte) (int, error) {
	if rw.logger != nil {
		rw.logger.Debug("Respond", "body", string(b))
	}
	return rw.ReadWriter.Write(b)
}

// NewLogReadWriter creates a new LogReadWriter.
func NewLogReadWriter(rw io.ReadWriter, logger *slog.Logger) *LogReadWriter {
	return &LogReadWriter{ReadWriter: rw, logger: logger}
}

type BufLogReadWriter struct {
	io.Writer
	*bufio.Reader
}

// NewBufLogReadWriter creates a new BufLogReadWriter. It wraps a bufio.ReadWriter and logs all reads and writes to a slog.Logger.
// the reason to divide it in 2 structs is to avoid the need to implement all the methods of bufio.ReadWriter
func NewBufLogReadWriter(rw io.ReadWriter, logger *slog.Logger) *BufLogReadWriter {
	rw = &LogReadWriter{ReadWriter: rw, logger: logger}

	return &BufLogReadWriter{
		Reader: bufio.NewReader(rw),
		Writer: rw,
	}
}

type HttpResponseWriter struct {
	http.ResponseWriter
	logger *slog.Logger
}

func (rw *HttpResponseWriter) Write(b []byte) (int, error) {
	if rw.logger != nil {
		rw.logger.Debug("Respond", "body", string(b))
	}
	return rw.ResponseWriter.Write(b)
}

func NewHttpResponseWriter(w http.ResponseWriter, logger *slog.Logger) *HttpResponseWriter {
	return &HttpResponseWriter{ResponseWriter: w, logger: logger}
}
