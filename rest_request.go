package dfetch

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// NewRESTRequest creates a JSON-oriented request builder for REST APIs.
func NewRESTRequest(c *Client, method Method, endpointPath string, args ...SetRequestOptionFn) *RESTRequest {
	return &RESTRequest{
		Id:           uuid.New().String(),
		client:       c,
		method:       method,
		endpointPath: endpointPath,
		options:      append([]SetRequestOptionFn(nil), args...),
	}
}

type RESTRequest struct {
	Id           string
	client       *Client
	method       Method
	endpointPath string
	options      []SetRequestOptionFn
}

func (rr *RESTRequest) AddOption(fn ...SetRequestOptionFn) *RESTRequest {
	rr.options = append(rr.options, fn...)
	return rr
}

func (rr *RESTRequest) AddHeader(args ...string) *RESTRequest {
	return rr.AddOption(AddHeader(args...))
}

func (rr *RESTRequest) AddQuery(args ...string) *RESTRequest {
	return rr.AddOption(AddQuery(args...))
}

func (rr *RESTRequest) Body(body interface{}) *RESTRequest {
	return rr.AddOption(SetJsonBody(body))
}

func (rr *RESTRequest) PreRequest(fn PreRequestFn) *RESTRequest {
	return rr.AddOption(PreRequest(fn))
}

// Do sends the request and unmarshals a JSON response into dst when dst is not nil.
func (rr *RESTRequest) Do(ctx context.Context, dst interface{}) (*http.Response, error) {
	rr.AddHeader("Accept", MimeTypeJson)
	ctx = context.WithValue(ctx, ContextRequestId, rr.Id)

	resp, respBody, err := rr.client.DoRequest(ctx, rr.method, rr.endpointPath, rr.options...)
	if err != nil {
		return nil, err
	}
	if dst == nil || len(respBody) == 0 {
		return resp, nil
	}
	if err := json.Unmarshal(respBody, dst); err != nil {
		return nil, err
	}
	return resp, nil
}
