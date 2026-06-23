package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	grpcfetch "github.com/digitalrealmforgestudios/d-fetch/grpc"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

const getUserMethod = "/example.UserService/GetUser"

type testUserService interface {
	GetUser(context.Context, *structpb.Struct) (*structpb.Struct, error)
}

type testServer struct{}

func (s testServer) GetUser(ctx context.Context, req *structpb.Struct) (*structpb.Struct, error) {
	if req.GetFields()["delay_ms"].GetNumberValue() > 0 {
		time.Sleep(time.Duration(req.GetFields()["delay_ms"].GetNumberValue()) * time.Millisecond)
	}

	md, _ := metadata.FromIncomingContext(ctx)
	resp, _ := structpb.NewStruct(map[string]interface{}{
		"id":            req.GetFields()["id"].GetStringValue(),
		"name":          "Jane Doe",
		"authorization": firstMetadata(md, "authorization"),
		"request_id":    firstMetadata(md, "x-request-id"),
	})
	return resp, nil
}

func TestInvoke(t *testing.T) {
	target, stop := startTestGRPCServer(t)
	defer stop()

	client, err := grpcfetch.NewClient(target, grpcfetch.Namespace("grpc-test"), grpcfetch.LogDump(true))
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}
	defer client.Close()

	req, _ := structpb.NewStruct(map[string]interface{}{"id": "USR-1001"})
	resp := new(structpb.Struct)

	err = client.Invoke(
		context.Background(),
		getUserMethod,
		req,
		resp,
		grpcfetch.AddMetadata("authorization", "Bearer token", "x-request-id", "REQ-1001"),
	)
	if err != nil {
		t.Fatalf("unexpected invoke error: %v", err)
	}
	if got := resp.GetFields()["authorization"].GetStringValue(); got != "Bearer token" {
		t.Fatalf("unexpected authorization metadata: %s", got)
	}
	if got := resp.GetFields()["request_id"].GetStringValue(); got != "REQ-1001" {
		t.Fatalf("unexpected request id metadata: %s", got)
	}
}

func TestInvokeTimeout(t *testing.T) {
	target, stop := startTestGRPCServer(t)
	defer stop()

	client, err := grpcfetch.NewClient(target, grpcfetch.Timeout(1000))
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}
	defer client.Close()

	req, _ := structpb.NewStruct(map[string]interface{}{"id": "USR-1001", "delay_ms": 100})
	err = client.Invoke(context.Background(), getUserMethod, req, new(structpb.Struct), grpcfetch.RequestTimeout(20))
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestInvalidAddMetadataArgs(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		err, ok := recovered.(error)
		if !ok || err.Error() != "grpcfetch: Invalid AddMetadata() args count must >= 2 and even" {
			t.Fatalf("unexpected panic: %#v", recovered)
		}
	}()
	client := grpcfetch.NewClientFromConn(nil)
	_ = client.Invoke(context.Background(), getUserMethod, nil, nil, grpcfetch.AddMetadata("key", "value", "dangling"))
}

func startTestGRPCServer(t *testing.T) (string, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := gogrpc.NewServer()
	registerTestService(server, testServer{})
	go func() {
		_ = server.Serve(lis)
	}()

	return lis.Addr().String(), func() {
		server.Stop()
		_ = lis.Close()
	}
}

func registerTestService(server *gogrpc.Server, svc testUserService) {
	server.RegisterService(&gogrpc.ServiceDesc{
		ServiceName: "example.UserService",
		HandlerType: (*testUserService)(nil),
		Methods: []gogrpc.MethodDesc{
			{
				MethodName: "GetUser",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor gogrpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(structpb.Struct)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(testUserService).GetUser(ctx, in)
					}
					info := &gogrpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: getUserMethod,
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(testUserService).GetUser(ctx, req.(*structpb.Struct))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
	}, svc)
}

func firstMetadata(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
