package httplog

import (
	"context"
	"net/http"
	"net/http/httputil"

	dlogger "github.com/digitalrealmforgestudios/d-logger"
	logOption "github.com/digitalrealmforgestudios/d-logger/option"
)

func DumpRequest(logger dlogger.Logger, ctx context.Context, enabled bool, req *http.Request, requestID string) {
	if !enabled {
		return
	}
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		logger.Warn("Unable to dump request. Error = %s", logOption.Format(err), logOption.Context(ctx))
		return
	}
	logger.Debug("\n---------- HTTP Request Dump -----------\n(RequestId=%s)\n%s\n----------------------------------------",
		logOption.Format(requestID, dump),
		logOption.Context(ctx),
	)
}

func DumpResponse(logger dlogger.Logger, ctx context.Context, enabled bool, resp *http.Response, requestID string) {
	if !enabled {
		return
	}
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		logger.Warn("Unable to dump response. Error = %s", logOption.Format(err), logOption.Context(ctx))
		return
	}
	logger.Debug("\n---------- HTTP Response Dump ----------\n(RequestId=%s)\n%s\n----------------------------------------",
		logOption.Format(requestID, dump),
		logOption.Context(ctx),
	)
}
