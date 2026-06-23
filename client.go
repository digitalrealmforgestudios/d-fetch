package dfetch

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/digitalrealmforgestudios/d-fetch/internal/body"
	"github.com/digitalrealmforgestudios/d-fetch/internal/httplog"
	"github.com/digitalrealmforgestudios/d-fetch/internal/requesturl"
	"github.com/digitalrealmforgestudios/d-fetch/internal/transport"
	dlogger "github.com/digitalrealmforgestudios/d-logger"
	logOption "github.com/digitalrealmforgestudios/d-logger/option"
	"github.com/google/uuid"
)

// TransporterOverrider can replace or wrap a client's RoundTripper.
type TransporterOverrider = transport.Overrider

// SetGlobalTransporterOverrider applies a RoundTripper override to clients created after this call.
func SetGlobalTransporterOverrider(fn TransporterOverrider) {
	transport.SetGlobalOverrider(fn)
}

func NewClient(baseUrl string, args ...SetClientOptionsFn) *Client {
	options := evaluateClientOptions(args)
	logger := dlogger.Get().NewChild(logOption.WithNamespace(options.namespace))
	httpClient := &http.Client{Transport: buildTransport(options.disableHTTP2)}

	if options.disableHTTP2 {
		logger.Debugf("HTTP/2 automatic switch is disabled")
	}

	httpClient.Transport = transport.ApplyGlobalOverrider(httpClient.Transport)

	return &Client{
		baseUrl:    baseUrl,
		httpClient: httpClient,
		log:        logger,
		logDump:    options.logDump,
	}
}

type Client struct {
	baseUrl    string
	httpClient *http.Client
	log        dlogger.Logger
	logDump    bool
}

func (c *Client) DoRequest(ctx context.Context, method Method, endpointPath string, args ...SetRequestOptionFn) (*http.Response, []byte, error) {
	options := evaluateRequestOptions(args)
	return c.doRequest(ctx, method, endpointPath, options)
}

func (c *Client) doRequest(ctx context.Context, method Method, endpointPath string, options requestOptions) (*http.Response, []byte, error) {
	if ctx == nil {
		return nil, nil, errors.New("d-fetch: ctx is required")
	}

	reqBody, err := body.Compose(string(method), options.headers[HeaderContentType], options.body)
	if err != nil {
		return nil, nil, err
	}
	if body.Unsupported(string(method), options.headers[HeaderContentType], options.body, reqBody) {
		c.log.Warn("Unsupported Content-Type in %s in request body", logOption.Format(options.headers[HeaderContentType]), logOption.Context(ctx))
	}

	reqCtx, cancel := withTimeout(ctx, options.timeoutMS)
	if cancel != nil {
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, method, requesturl.Join(c.baseUrl, endpointPath, options.query), bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil, err
	}
	applyHeaders(req, options)

	if options.preRequest != nil {
		options.preRequest(req, reqBody)
	}

	requestID := c.requestID(ctx)
	startedAt := time.Now()
	httplog.DumpRequest(c.log, ctx, c.logDump, req, requestID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("HTTP Request  (Id=%s) Failed to do request", logOption.Format(requestID), logOption.Error(err), logOption.Context(ctx))
		return nil, nil, err
	}
	httplog.DumpResponse(c.log, ctx, c.logDump, resp, requestID)

	respBody, err := readAndClose(c.log, ctx, resp, requestID)
	if err != nil {
		return nil, nil, err
	}

	c.log.Debug("HTTP Request  (Id=%s) URL=\"%s %s\" ResponseStatus=\"%s\" TimeElapsed=\"%s\"",
		logOption.Format(requestID, req.Method, req.URL.String(), resp.Status, time.Since(startedAt)),
		logOption.Context(ctx),
	)
	return resp, respBody, nil
}

func buildTransport(disableHTTP2 bool) http.RoundTripper {
	if !disableHTTP2 {
		return nil
	}
	return &http.Transport{
		TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
	}
}

func withTimeout(ctx context.Context, timeoutMS int) (context.Context, context.CancelFunc) {
	if timeoutMS <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
}

func applyHeaders(req *http.Request, options requestOptions) {
	for key, value := range options.headers {
		if options.canonicalHeader {
			req.Header.Set(key, value)
			continue
		}
		req.Header[key] = []string{value}
	}
}

func readAndClose(logger dlogger.Logger, ctx context.Context, resp *http.Response, requestID string) ([]byte, error) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warn("HTTP Response (Id=%s) Failed to close Body reader. Error = %s",
				logOption.Format(requestID, err),
				logOption.Context(ctx),
			)
		}
	}()
	return io.ReadAll(resp.Body)
}

func (c *Client) requestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(ContextRequestId).(string); ok {
		return reqID
	}
	return uuid.New().String()
}
