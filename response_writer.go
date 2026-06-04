package limen

import (
	"bytes"
	"net/http"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool

	// Deferred response fields (only used when deferWrite=true)
	deferWrite      bool // Only true when after hooks exist
	payload         any  // Raw payload (before envelope transformation)
	isError         bool // True if Error() was called (vs JSON())
	written         bool // True if Responder stored data
	modified        bool // True if hook modified response
	modifiedPayload any  // Modified payload from hook
	modifiedStatus  int  // Modified status from hook

	redirectURL    string
	redirectStatus int

	directBody    bytes.Buffer
	directWritten bool

	// Auth result stored for hooks to access
	authResult *AuthenticationResult
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		if !rw.deferWrite {
			rw.ResponseWriter.WriteHeader(code)
		}
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	if rw.deferWrite {
		rw.directWritten = true
		return rw.directBody.Write(b)
	}
	return rw.ResponseWriter.Write(b)
}
