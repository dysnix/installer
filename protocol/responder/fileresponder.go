package responder

import (
	"net/http"
	"strconv"

	"github.com/go-openapi/runtime"
)

// FileResponder is a file responce wrapper
// to be integrated with swagger generated code
type FileResponder struct {
	name     string
	contType string
	content  []byte
}

// NewFileResponder creates a file responder
func NewFileResponder(name string, contType string, content []byte) *FileResponder {
	return &FileResponder{name: name, contType: contType, content: content}
}

// WriteResponse is an actual responce write function
func (r *FileResponder) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {
	rw.Header().Set("Content-Type", r.contType)
	rw.Header().Set("Content-Length", strconv.Itoa(len(r.content)))
	rw.Header().Set("Content-Disposition", `attachment; filename="`+r.name+`"`)
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "POST, PUT, GET, OPTIONS, DELETE")
	rw.Header().Set("Access-Control-Allow-Headers", "X-Api-Key, Content-Type")

	rw.WriteHeader(http.StatusOK)

	if _, err := rw.Write(r.content); err != nil {
		panic(err)
	}
}
