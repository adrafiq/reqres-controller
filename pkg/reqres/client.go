package reqres

import (
	"net/http"

	"github.com/go-logr/logr"
)

type Client struct {
	HTTPClient *http.Client
	HostUrl    string
	logger     *logr.Logger
}

func NewClient(host string, logger *logr.Logger) Client {
	client := Client{
		HostUrl:    host,
		HTTPClient: &http.Client{},
		logger:     logger,
	}
	return client
}
