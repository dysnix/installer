package responder

import (
	"net/http"

	"git.arilot.com/kuberstack/kuberstack-installer/protocol/gen/models"

	"github.com/go-openapi/runtime"
)

// Responder is a responce wrapper
// to be integrated with swagger generated code
type Responder struct {
	code     int
	response interface{}
	headers  http.Header
}

// OK creates a StatusOK responder
func OK(response interface{}) *Responder {
	return &Responder{
		http.StatusOK,
		response,
		make(http.Header),
	}
}

// File creates a file responder
func File(response interface{}) *Responder {
	return &Responder{
		http.StatusOK,
		response,
		make(http.Header),
	}
}

// SimpleOK creates a StatusOK responder with no payload
func SimpleOK() *Responder {
	return &Responder{
		http.StatusOK,
		&models.StatusResponse{Status: true},
		make(http.Header),
	}
}

// NoContent creates a StatusNoContent responder
func NoContent() *Responder {
	return &Responder{
		http.StatusNoContent,
		nil,
		make(http.Header),
	}
}

// NotOK creates a StatusOK with Status=false responder
func NotOK(message string) *Responder {
	return &Responder{
		http.StatusOK,
		&models.StatusResponse{Status: false, Message: models.StatusMessage(message)},
		make(http.Header),
	}
}

// Err500 creates a InternalSererError responder
func Err500(err error) *Responder {
	return &Responder{
		http.StatusInternalServerError,
		err,
		make(http.Header),
	}
}

// WriteResponse is an actual response write function
func (r *Responder) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {
	for k, v := range r.headers {
		for _, val := range v {
			rw.Header().Add(k, val)
		}
	}

	rw.WriteHeader(r.code)

	if r.response != nil {
		if err := producer.Produce(rw, r.response); err != nil {
			panic(err)
		}
	}
}
