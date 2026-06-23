package dfetch_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	dfetch "github.com/digitalrealmforgestudios/d-fetch"
)

type echoResult struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Header map[string][]string `json:"header"`
	Query  url.Values          `json:"query"`
	Body   string              `json:"body"`
	JSON   map[string]string   `json:"json"`
	Form   url.Values          `json:"form"`
}

func newTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/xml" {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte("<ok/>"))
			return
		}
		if r.URL.Path == "/delay" {
			time.Sleep(200 * time.Millisecond)
		}

		result := echoResult{
			Method: r.Method,
			URL:    r.URL.String(),
			Header: r.Header,
			Query:  r.URL.Query(),
			Form:   make(url.Values),
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		result.Body = string(bodyBytes)
		if r.Header.Get(dfetch.HeaderContentType) == dfetch.MimeTypeJson && len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &result.JSON)
		}
		if r.Header.Get(dfetch.HeaderContentType) == dfetch.MimeTypeUrlEncodedForm && len(bodyBytes) > 0 {
			result.Form, _ = url.ParseQuery(string(bodyBytes))
		}
		w.Header().Set("Content-Type", dfetch.MimeTypeJson)
		_ = json.NewEncoder(w).Encode(result)
	}))
}

func TestTimeout(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	client := dfetch.NewClient(srv.URL)
	_, _, err := client.DoRequest(context.Background(), dfetch.MethodGet, "/delay", dfetch.Timeout(50))
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestInvalidJsonBody(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	client := dfetch.NewClient(srv.URL)
	_, _, err := client.DoRequest(context.Background(), dfetch.MethodPost, "/", dfetch.SetJsonBody(json.RawMessage("{")))
	if err == nil || err.Error() != `d-fetch: Failed to compose request body. ContentType = application/json, Error = json: error calling MarshalJSON for type json.RawMessage: unexpected end of JSON input` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNilContext(t *testing.T) {
	client := dfetch.NewClient("http://example.test")
	_, _, err := client.DoRequest(nil, dfetch.MethodPost, "/")
	if err == nil || err.Error() != `d-fetch: ctx is required` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvalidAddHeaderArgs(t *testing.T) {
	assertPanicError(t, `d-fetch: Invalid AddHeader() args count must >= 2 and even`, func() {
		dfetch.NewClient("http://example.test").DoRequest(context.Background(), dfetch.MethodPost, "/", dfetch.AddHeader("key", "value", "dangling"))
	})
}

func TestInvalidAddQueryArgs(t *testing.T) {
	assertPanicError(t, `d-fetch: Invalid AddQuery() args count must >= 2 and even`, func() {
		dfetch.NewClient("http://example.test").DoRequest(context.Background(), dfetch.MethodPost, "/", dfetch.AddQuery("key", "value", "dangling"))
	})
}

func TestBodies(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	client := dfetch.NewClient(srv.URL)

	tests := []struct {
		name   string
		option dfetch.SetRequestOptionFn
		check  func(t *testing.T, got echoResult)
	}{
		{name: "nil json", option: dfetch.SetJsonBody(nil), check: func(t *testing.T, got echoResult) {
			if got.Body != "" {
				t.Fatalf("expected empty body, got %q", got.Body)
			}
		}},
		{name: "raw bytes", option: dfetch.SetBody([]byte("hello")), check: func(t *testing.T, got echoResult) {
			if got.Body != "hello" {
				t.Fatalf("expected raw body, got %q", got.Body)
			}
		}},
		{name: "json", option: dfetch.SetJsonBody(map[string]string{"message": "hello"}), check: func(t *testing.T, got echoResult) {
			if got.JSON["message"] != "hello" {
				t.Fatalf("expected json body, got %#v", got.JSON)
			}
		}},
		{name: "form", option: dfetch.SetUrlEncodedFormBody(url.Values{"message": []string{"hello"}}), check: func(t *testing.T, got echoResult) {
			if got.Form.Get("message") != "hello" {
				t.Fatalf("expected form body, got %#v", got.Form)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, respBody, err := client.DoRequest(context.Background(), dfetch.MethodPost, "/", tt.option)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var got echoResult
			if err := json.Unmarshal(respBody, &got); err != nil {
				t.Fatalf("unexpected json: %v", err)
			}
			tt.check(t, got)
		})
	}
}

func TestInvalidUrlEncodedFormBody(t *testing.T) {
	client := dfetch.NewClient("http://example.test")
	_, _, err := client.DoRequest(context.Background(), dfetch.MethodPost, "/",
		dfetch.AddHeader(dfetch.HeaderContentType, dfetch.MimeTypeUrlEncodedForm),
		dfetch.SetBody("invalid"),
	)
	if err == nil || err.Error() != "d-fetch: Unable to compose URL-Encoded Form, body is not url.Values type. Type = string" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOverrideTransporter(t *testing.T) {
	dfetch.SetGlobalTransporterOverrider(func(_ http.RoundTripper) http.RoundTripper {
		return roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "204 No Content",
				StatusCode: http.StatusNoContent,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		})
	})
	defer dfetch.SetGlobalTransporterOverrider(nil)

	client := dfetch.NewClient("http://example.test")
	resp, _, err := client.DoRequest(context.Background(), dfetch.MethodHead, "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func assertPanicError(t *testing.T, expected string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		err, ok := recovered.(error)
		if !ok || err.Error() != expected {
			t.Fatalf("unexpected panic: %#v", recovered)
		}
	}()
	fn()
}
