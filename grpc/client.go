package grpc

import (
	"context"
	"time"

	dlogger "github.com/digitalrealmforgestudios/d-logger"
	logOption "github.com/digitalrealmforgestudios/d-logger/option"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	target  string
	conn    *grpc.ClientConn
	log     dlogger.Logger
	logDump bool
	timeout time.Duration
}

func NewClient(target string, args ...SetClientOptionsFn) (*Client, error) {
	options := evaluateClientOptions(args)
	conn, err := grpc.NewClient(target, options.dialOptions...)
	if err != nil {
		return nil, err
	}

	return &Client{
		target:  target,
		conn:    conn,
		log:     dlogger.Get().NewChild(logOption.WithNamespace(options.namespace)),
		logDump: options.logDump,
		timeout: options.timeout,
	}, nil
}

// NewClientFromConn wraps an existing gRPC connection.
func NewClientFromConn(conn *grpc.ClientConn, args ...SetClientOptionsFn) *Client {
	options := evaluateClientOptions(args)
	return &Client{
		conn:    conn,
		log:     dlogger.Get().NewChild(logOption.WithNamespace(options.namespace)),
		logDump: options.logDump,
		timeout: options.timeout,
	}
}

func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) Invoke(ctx context.Context, method string, request interface{}, response interface{}, args ...SetRequestOptionsFn) error {
	options := evaluateRequestOptions(c.timeout, args)
	callCtx, cancel := prepareContext(ctx, options)
	if cancel != nil {
		defer cancel()
	}

	if options.preRequest != nil {
		callCtx = options.preRequest(callCtx, method, request)
	}
	if callCtx == nil {
		callCtx = context.Background()
	}

	requestID := requestIDFromContext(callCtx)
	startedAt := time.Now()
	DumpRequest(c.log, callCtx, c.logDump, requestID, c.target, method, outgoingMetadata(callCtx), request)

	err := c.conn.Invoke(callCtx, method, request, response, options.callOptions...)
	if err != nil {
		c.log.Error("gRPC Request (Id=%s) Method=\"%s\" Failed to invoke",
			logOption.Format(requestID, method),
			logOption.Error(err),
			logOption.Context(callCtx),
		)
		return err
	}

	DumpResponse(c.log, callCtx, c.logDump, requestID, method, response)
	c.log.Debug("gRPC Request (Id=%s) Method=\"%s\" TimeElapsed=\"%s\"",
		logOption.Format(requestID, method, time.Since(startedAt)),
		logOption.Context(callCtx),
	)
	return nil
}

func outgoingMetadata(ctx context.Context) metadata.MD {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return nil
	}
	return md
}

func prepareContext(ctx context.Context, options requestOptions) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(options.metadata) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, options.metadata)
	}
	if options.timeout <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, options.timeout)
}

func requestIDFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(ContextRequestId).(string); ok && value != "" {
		return value
	}
	return uuid.New().String()
}
