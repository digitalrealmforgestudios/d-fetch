# d-fetch

Small HTTP client helper for Go. The public API keeps the familiar `http fetch use net htttp` style while the module is published as:

```shell
go get github.com/digitalrealmforgestudios/d-fetch
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	dfetch "github.com/digitalrealmforgestudios/d-fetch"
)

func main() {
	client := dfetch.NewClient("https://api.example.com")

	resp, body, err := client.DoRequest(
		context.Background(),
		dfetch.MethodGet,
		"/users",
		dfetch.AddQuery("page", "1"),
		dfetch.AddHeader("Accept", dfetch.MimeTypeJson),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.StatusCode, string(body))
}
```

## REST JSON Builder

```go
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

client := dfetch.NewClient("https://api.example.com")
req := dfetch.NewRESTRequest(client, dfetch.MethodPost, "/users").
	Body(map[string]string{"name": "Jane"})

var user User
_, err := req.Do(context.Background(), &user)
```

## Options

- `Namespace(name)` changes the logger namespace.
- `LogDump(true)` logs raw HTTP request and response dumps.
- `DisableHTTP2()` disables automatic HTTP/2 transport upgrade.
- `AddHeader(key, value, ...)` adds request headers.
- `AddQuery(key, value, ...)` adds query parameters.
- `SetBody(body)` sends raw `[]byte` bodies.
- `SetJsonBody(body)` sends JSON.
- `SetUrlEncodedFormBody(url.Values)` sends form bodies.
- `Timeout(ms)` sets request timeout in milliseconds.
- `PreRequest(fn)` mutates the request before it is sent.
- `DisableCanonicalHeader()` keeps header keys exactly as assigned on the outgoing request object.

## Transport Override

Use `SetGlobalTransporterOverrider` to wrap clients created after setup, for example with OpenTelemetry:

```go
func init() {
	dfetch.SetGlobalTransporterOverrider(func(existing http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(existing)
	})
}
```

## Logging

Request and response dumps use `github.com/digitalrealmforgestudios/d-logger`.

```go
client := dfetch.NewClient(
	"https://api.example.com",
	dfetch.Namespace("api-client"),
	dfetch.LogDump(true),
)
```

## Runnable Example

Run the complete client example:

```shell
go run ./examples/client
```

The example initializes `d-logger`, starts a local test API, and demonstrates:

- basic `DoRequest`
- JSON REST builder
- URL encoded form body
- request mutation with `PreRequest`
- timeout handling
- request and response dump logs

## gRPC Client

The module also provides a gRPC client in the `/grpc` subpackage:

```go
import grpcfetch "github.com/digitalrealmforgestudios/d-fetch/grpc"
```

```go
client, err := grpcfetch.NewClient(
	"localhost:50051",
	grpcfetch.Namespace("user-grpc-client"),
	grpcfetch.LogDump(true),
	grpcfetch.Timeout(3000),
)
if err != nil {
	panic(err)
}
defer client.Close()

err = client.Invoke(
	context.Background(),
	"/user.UserService/GetUser",
	&pb.GetUserRequest{Id: "USR-1001"},
	&pb.GetUserResponse{},
	grpcfetch.AddMetadata("authorization", "Bearer token"),
	grpcfetch.RequestTimeout(1000),
)
```

Run the complete gRPC example:

```shell
go run ./examples/grpc
```

The gRPC client supports:

- unary `Invoke`
- request metadata
- per-client and per-request timeout
- `PreRequest` context hook
- TLS or insecure transport credentials
- request and response dump logs through `d-logger`
