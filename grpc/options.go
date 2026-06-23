package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type SetClientOptionsFn func(*clientOptions)

type clientOptions struct {
	namespace   string
	logDump     bool
	timeout     time.Duration
	creds       credentials.TransportCredentials
	dialOptions []gogrpc.DialOption
}

func defaultClientOptions() clientOptions {
	return clientOptions{
		namespace: "grpc",
		timeout:   10 * time.Second,
		creds:     insecure.NewCredentials(),
	}
}

// Namespace changes the logger namespace used by a gRPC client.
func Namespace(namespace string) SetClientOptionsFn {
	return func(o *clientOptions) {
		o.namespace = namespace
	}
}

// LogDump enables or disables gRPC request and response dumps.
func LogDump(enable bool) SetClientOptionsFn {
	return func(o *clientOptions) {
		o.logDump = enable
	}
}

// Timeout changes the default per-call timeout in milliseconds.
func Timeout(ms int) SetClientOptionsFn {
	return func(o *clientOptions) {
		o.timeout = durationFromMilliseconds(ms)
	}
}

// WithInsecure configures the client to use an insecure transport.
func WithInsecure() SetClientOptionsFn {
	return func(o *clientOptions) {
		o.creds = insecure.NewCredentials()
	}
}

// WithTLS configures the client to use TLS transport credentials.
func WithTLS(config *tls.Config) SetClientOptionsFn {
	return WithTransportCredentials(credentials.NewTLS(config))
}

// WithTransportCredentials configures custom transport credentials.
func WithTransportCredentials(creds credentials.TransportCredentials) SetClientOptionsFn {
	return func(o *clientOptions) {
		if creds != nil {
			o.creds = creds
		}
	}
}

// WithDialOption passes a grpc.DialOption to the underlying gRPC client.
func WithDialOption(option gogrpc.DialOption) SetClientOptionsFn {
	return func(o *clientOptions) {
		if option != nil {
			o.dialOptions = append(o.dialOptions, option)
		}
	}
}

func evaluateClientOptions(fns []SetClientOptionsFn) clientOptions {
	options := defaultClientOptions()
	for _, fn := range fns {
		if fn != nil {
			fn(&options)
		}
	}
	if options.creds != nil {
		options.dialOptions = append([]gogrpc.DialOption{gogrpc.WithTransportCredentials(options.creds)}, options.dialOptions...)
	}
	return options
}

type SetRequestOptionsFn func(*requestOptions)

// PreRequestFn can inspect and replace the outgoing context before Invoke.
type PreRequestFn func(ctx context.Context, method string, request interface{}) context.Context

type requestOptions struct {
	metadata    metadata.MD
	timeout     time.Duration
	preRequest  PreRequestFn
	callOptions []gogrpc.CallOption
}

func defaultRequestOptions(defaultTimeout time.Duration) requestOptions {
	return requestOptions{
		metadata: make(metadata.MD),
		timeout:  defaultTimeout,
	}
}

// AddMetadata appends outgoing gRPC metadata.
func AddMetadata(args ...string) SetRequestOptionsFn {
	return func(o *requestOptions) {
		requirePairs("AddMetadata", args)
		for i := 0; i < len(args); i += 2 {
			o.metadata.Append(args[i], args[i+1])
		}
	}
}

// SetMetadata replaces outgoing gRPC metadata.
func SetMetadata(md metadata.MD) SetRequestOptionsFn {
	return func(o *requestOptions) {
		o.metadata = md.Copy()
	}
}

// RequestTimeout overrides the timeout for one Invoke call.
func RequestTimeout(ms int) SetRequestOptionsFn {
	return func(o *requestOptions) {
		o.timeout = durationFromMilliseconds(ms)
	}
}

// PreRequest registers a hook that runs after metadata and timeout are applied.
func PreRequest(fn PreRequestFn) SetRequestOptionsFn {
	return func(o *requestOptions) {
		o.preRequest = fn
	}
}

// WithCallOption passes a grpc.CallOption to one Invoke call.
func WithCallOption(option gogrpc.CallOption) SetRequestOptionsFn {
	return func(o *requestOptions) {
		if option != nil {
			o.callOptions = append(o.callOptions, option)
		}
	}
}

func evaluateRequestOptions(defaultTimeout time.Duration, fns []SetRequestOptionsFn) requestOptions {
	options := defaultRequestOptions(defaultTimeout)
	for _, fn := range fns {
		if fn != nil {
			fn(&options)
		}
	}
	return options
}

func requirePairs(name string, args []string) {
	if len(args) == 0 || len(args)%2 == 1 {
		panic(errors.New("grpcfetch: Invalid " + name + "() args count must >= 2 and even"))
	}
}

func durationFromMilliseconds(ms int) time.Duration {
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}
