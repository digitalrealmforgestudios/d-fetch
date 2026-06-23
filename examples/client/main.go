package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	dfetch "github.com/digitalrealmforgestudios/d-fetch"
	dlogger "github.com/digitalrealmforgestudios/d-logger"
)

type user struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type apiResponse struct {
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	RequestID     string            `json:"requestId,omitempty"`
	Authorization string            `json:"authorization,omitempty"`
	Query         map[string]string `json:"query,omitempty"`
	Form          map[string]string `json:"form,omitempty"`
	User          *user             `json:"user,omitempty"`
	Message       string            `json:"message,omitempty"`
}

func main() {
	initLogger()

	server := newExampleAPIServer()
	defer server.Close()

	client := dfetch.NewClient(
		server.URL,
		dfetch.Namespace("example-api-client"),
		dfetch.LogDump(true),
	)

	ctx := context.Background()

	if err := runBasicRequest(ctx, client); err != nil {
		exitWithError("basic request", err)
	}
	if err := runRESTBuilder(ctx, client); err != nil {
		exitWithError("rest builder", err)
	}
	if err := runFormRequest(ctx, client); err != nil {
		exitWithError("form request", err)
	}
	if err := runPreRequestHook(ctx, client); err != nil {
		exitWithError("pre request hook", err)
	}
	if err := runTimeout(ctx, client); err != nil {
		exitWithError("timeout", err)
	}
}

func initLogger() {
	_ = os.Setenv(dlogger.EnvServiceName, "d-fetch-example")
	_ = os.Setenv(dlogger.EnvServiceVersion, "0.1.0")
	_ = os.Setenv(dlogger.EnvDeploymentEnvironment, "local")

	dlogger.Register(dlogger.New("d-fetch-example", "debug", os.Stdout))
}

func runBasicRequest(ctx context.Context, client *dfetch.Client) error {
	resp, body, err := client.DoRequest(
		ctx,
		dfetch.MethodGet,
		"/users",
		dfetch.AddHeader("Accept", dfetch.MimeTypeJson),
		dfetch.AddQuery("page", "1", "limit", "10"),
		dfetch.Timeout(2000),
	)
	if err != nil {
		return err
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	fmt.Printf("basic request: status=%d path=%s page=%s\n", resp.StatusCode, result.Path, result.Query["page"])
	return nil
}

func runRESTBuilder(ctx context.Context, client *dfetch.Client) error {
	req := dfetch.NewRESTRequest(client, dfetch.MethodPost, "/users").
		AddHeader("X-Team", "platform").
		Body(user{
			Name:  "Jane Doe",
			Email: "jane@example.com",
		})

	var result apiResponse
	resp, err := req.Do(ctx, &result)
	if err != nil {
		return err
	}

	fmt.Printf("rest builder: status=%d created_user=%s id=%s\n", resp.StatusCode, result.User.Name, result.User.ID)
	return nil
}

func runFormRequest(ctx context.Context, client *dfetch.Client) error {
	form := url.Values{}
	form.Set("username", "jane")
	form.Set("password", "secret")

	_, body, err := client.DoRequest(
		ctx,
		dfetch.MethodPost,
		"/login",
		dfetch.SetUrlEncodedFormBody(form),
	)
	if err != nil {
		return err
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	fmt.Printf("form request: message=%s username=%s\n", result.Message, result.Form["username"])
	return nil
}

func runPreRequestHook(ctx context.Context, client *dfetch.Client) error {
	_, body, err := client.DoRequest(
		ctx,
		dfetch.MethodGet,
		"/secure",
		dfetch.PreRequest(func(req *http.Request, _ []byte) {
			req.Header.Set("Authorization", "Bearer example-token")
			req.Header.Set("X-Request-ID", "REQ-1001")
		}),
	)
	if err != nil {
		return err
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	fmt.Printf("pre request hook: auth=%s request_id=%s\n", result.Authorization, result.RequestID)
	return nil
}

func runTimeout(ctx context.Context, client *dfetch.Client) error {
	_, _, err := client.DoRequest(
		ctx,
		dfetch.MethodGet,
		"/slow",
		dfetch.Timeout(50),
	)
	if err == nil {
		return errors.New("expected timeout error")
	}

	fmt.Printf("timeout: received expected error=%v\n", err)
	return nil
}

func newExampleAPIServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, apiResponse{
				Method: r.Method,
				Path:   r.URL.Path,
				Query:  singleValueQuery(r.URL.Query()),
			})
		case http.MethodPost:
			var payload user
			_ = json.NewDecoder(r.Body).Decode(&payload)
			payload.ID = "USR-1001"
			writeJSON(w, http.StatusCreated, apiResponse{
				Method: r.Method,
				Path:   r.URL.Path,
				User:   &payload,
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		writeJSON(w, http.StatusOK, apiResponse{
			Method:  r.Method,
			Path:    r.URL.Path,
			Form:    singleValueQuery(r.PostForm),
			Message: "login accepted",
		})
	})

	mux.HandleFunc("/secure", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, apiResponse{
			Method:        r.Method,
			Path:          r.URL.Path,
			RequestID:     r.Header.Get("X-Request-ID"),
			Authorization: r.Header.Get("Authorization"),
		})
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		writeJSON(w, http.StatusOK, apiResponse{Message: "slow response"})
	})

	return httptest.NewServer(mux)
}

func singleValueQuery(values url.Values) map[string]string {
	result := make(map[string]string, len(values))
	for key := range values {
		result[key] = values.Get(key)
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", dfetch.MimeTypeJson)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func exitWithError(name string, err error) {
	fmt.Fprintf(os.Stderr, "%s failed: %v\n", name, err)
	os.Exit(1)
}
