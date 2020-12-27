package common

import (
	"log"
)

// Logger holds several logger instances with different prefixes
type Logger struct {
	Warn *log.Logger
	Info *log.Logger
	Err  *log.Logger
}

// ResponseRecord holds a single neighbor
type ResponseRecord struct {
	ID   string  `json:"id,omitempty"`
	Dist float64 `json:"dist,omitempty"`
}

// ResponseData holds the resulting objectIDs of nearest neighbors found
type ResponseData struct {
	Neighbors []ResponseRecord `json:"neighbors,omitempty"`
	Message   string           `json:"message,omitempty"`
}

// RequestData used for unpacking the request payload for Pop/Put vectors
type RequestData struct {
	ID  string    `json:"id,omitempty"`
	Vec []float64 `json:"vec,omitempty"`
}
