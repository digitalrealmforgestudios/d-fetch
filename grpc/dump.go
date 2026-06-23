package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	dlogger "github.com/digitalrealmforgestudios/d-logger"
	logOption "github.com/digitalrealmforgestudios/d-logger/option"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func DumpRequest(logger dlogger.Logger, ctx context.Context, enabled bool, requestID string, target string, method string, md metadata.MD, body interface{}) {
	if !enabled {
		return
	}
	logger.Debug("\n---------- gRPC Request Dump -----------\n(RequestId=%s)\nTarget: %s\nMethod: %s\nMetadata:\n%s\nBody:\n%s\n----------------------------------------",
		logOption.Format(requestID, target, method, formatMetadata(md), formatMessage(body)),
		logOption.Context(ctx),
	)
}

func DumpResponse(logger dlogger.Logger, ctx context.Context, enabled bool, requestID string, method string, body interface{}) {
	if !enabled {
		return
	}
	logger.Debug("\n---------- gRPC Response Dump ----------\n(RequestId=%s)\nMethod: %s\nBody:\n%s\n----------------------------------------",
		logOption.Format(requestID, method, formatMessage(body)),
		logOption.Context(ctx),
	)
}

func formatMetadata(md metadata.MD) string {
	if len(md) == 0 {
		return "{}"
	}
	data, err := json.MarshalIndent(map[string][]string(md), "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", md)
	}
	return string(data)
}

func formatMessage(msg interface{}) string {
	if msg == nil {
		return "<nil>"
	}
	if pb, ok := msg.(proto.Message); ok {
		data, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(pb)
		if err == nil {
			return string(data)
		}
	}
	data, err := json.MarshalIndent(msg, "", "  ")
	if err == nil {
		return string(data)
	}
	return strings.TrimSpace(fmt.Sprintf("%#v", msg))
}
