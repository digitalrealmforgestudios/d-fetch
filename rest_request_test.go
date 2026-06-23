package dfetch_test

import (
	"context"
	"net/http"
	"testing"

	dfetch "github.com/digitalrealmforgestudios/d-fetch"
)

func TestRestGet(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	client := dfetch.NewClient(srv.URL)
	req := dfetch.NewRESTRequest(client, dfetch.MethodGet, "/anything").AddQuery("message", "hello")

	var got echoResult
	_, err := req.Do(context.Background(), &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Method != http.MethodGet || got.Query.Get("message") != "hello" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestRestPostBody(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	client := dfetch.NewClient(srv.URL)
	req := dfetch.NewRESTRequest(client, dfetch.MethodPost, "/anything").
		Body(map[string]string{"message": "hello"})

	var got echoResult
	_, err := req.Do(context.Background(), &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.JSON["message"] != "hello" {
		t.Fatalf("unexpected json body: %#v", got.JSON)
	}
}

func TestRestSkipResponseBodyParsing(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	client := dfetch.NewClient(srv.URL)
	_, err := dfetch.NewRESTRequest(client, dfetch.MethodPost, "/anything").
		Body(map[string]string{"message": "hello"}).
		Do(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestXMLResponse(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	client := dfetch.NewClient(srv.URL)
	_, err := dfetch.NewRESTRequest(client, dfetch.MethodGet, "/xml").Do(context.Background(), &echoResult{})
	if err == nil || err.Error() != `invalid character '<' looking for beginning of value` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestPreRequestAndCanonicalHeader(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	client := dfetch.NewClient(srv.URL, dfetch.Namespace("test"), dfetch.LogDump(true))
	headerWasPreserved := false
	req := dfetch.NewRESTRequest(client, dfetch.MethodGet, "/anything", dfetch.DisableCanonicalHeader()).
		AddHeader("MESSAGE", "hello").
		PreRequest(func(r *http.Request, _ []byte) {
			headerWasPreserved = len(r.Header["MESSAGE"]) == 1 && r.Header["MESSAGE"][0] == "hello"
			r.URL.RawQuery = "signature=yes"
		})

	var got echoResult
	_, err := req.Do(context.Background(), &got)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !headerWasPreserved || got.Query.Get("signature") != "yes" {
		t.Fatalf("unexpected response: %#v", got)
	}
}
