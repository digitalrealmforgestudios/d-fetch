package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	grpcfetch "github.com/digitalrealmforgestudios/d-fetch/grpc"
	dlogger "github.com/digitalrealmforgestudios/d-logger"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

const getUserMethod = "/example.UserService/GetUser"

func main() {
	initLogger()

	target, stop, err := startExampleGRPCServer()
	if err != nil {
		exitWithError("start server", err)
	}
	defer stop()

	client, err := grpcfetch.NewClient(
		target,
		grpcfetch.Namespace("example-grpc-client"),
		grpcfetch.LogDump(true),
		grpcfetch.Timeout(2000),
	)
	if err != nil {
		exitWithError("create client", err)
	}
	defer client.Close()

	if err := runUnaryInvoke(context.Background(), client); err != nil {
		exitWithError("unary invoke", err)
	}
	if err := runPreRequest(context.Background(), client); err != nil {
		exitWithError("pre request", err)
	}
	if err := runTimeout(context.Background(), client); err != nil {
		exitWithError("timeout", err)
	}
}

func initLogger() {
	_ = os.Setenv(dlogger.EnvServiceName, "d-fetch-grpc-example")
	_ = os.Setenv(dlogger.EnvServiceVersion, "0.1.0")
	_ = os.Setenv(dlogger.EnvDeploymentEnvironment, "local")

	dlogger.Register(dlogger.New("d-fetch-grpc-example", "debug", os.Stdout))
}

func runUnaryInvoke(ctx context.Context, client *grpcfetch.Client) error {
	req, _ := structpb.NewStruct(map[string]interface{}{"id": "USR-1001"})
	resp := new(structpb.Struct)

	err := client.Invoke(
		ctx,
		getUserMethod,
		req,
		resp,
		grpcfetch.AddMetadata("authorization", "Bearer example-token", "x-request-id", "REQ-1001"),
	)
	if err != nil {
		return err
	}

	fmt.Printf("grpc unary: id=%s name=%s request_id=%s\n",
		resp.GetFields()["id"].GetStringValue(),
		resp.GetFields()["name"].GetStringValue(),
		resp.GetFields()["request_id"].GetStringValue(),
	)
	return nil
}

func runPreRequest(ctx context.Context, client *grpcfetch.Client) error {
	req, _ := structpb.NewStruct(map[string]interface{}{"id": "USR-2002"})
	resp := new(structpb.Struct)

	err := client.Invoke(
		ctx,
		getUserMethod,
		req,
		resp,
		grpcfetch.PreRequest(func(ctx context.Context, method string, request interface{}) context.Context {
			return metadata.AppendToOutgoingContext(ctx, "x-request-id", "REQ-PRE-2002")
		}),
	)
	if err != nil {
		return err
	}

	fmt.Printf("grpc pre request: method=%s request_id=%s\n",
		getUserMethod,
		resp.GetFields()["request_id"].GetStringValue(),
	)
	return nil
}

func runTimeout(ctx context.Context, client *grpcfetch.Client) error {
	req, _ := structpb.NewStruct(map[string]interface{}{"id": "USR-3003", "delay_ms": 150})
	err := client.Invoke(ctx, getUserMethod, req, new(structpb.Struct), grpcfetch.RequestTimeout(50))
	if err == nil {
		return fmt.Errorf("expected timeout error")
	}

	fmt.Printf("grpc timeout: received expected error=%v\n", err)
	return nil
}

type userService interface {
	GetUser(context.Context, *structpb.Struct) (*structpb.Struct, error)
}

type exampleUserServer struct{}

func (s exampleUserServer) GetUser(ctx context.Context, req *structpb.Struct) (*structpb.Struct, error) {
	if delay := req.GetFields()["delay_ms"].GetNumberValue(); delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
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

func startExampleGRPCServer() (string, func(), error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	server := gogrpc.NewServer()
	registerUserService(server, exampleUserServer{})
	go func() {
		_ = server.Serve(lis)
	}()

	return lis.Addr().String(), func() {
		server.Stop()
		_ = lis.Close()
	}, nil
}

func registerUserService(server *gogrpc.Server, svc userService) {
	server.RegisterService(&gogrpc.ServiceDesc{
		ServiceName: "example.UserService",
		HandlerType: (*userService)(nil),
		Methods: []gogrpc.MethodDesc{
			{
				MethodName: "GetUser",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor gogrpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(structpb.Struct)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(userService).GetUser(ctx, in)
					}
					info := &gogrpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: getUserMethod,
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(userService).GetUser(ctx, req.(*structpb.Struct))
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

func exitWithError(name string, err error) {
	fmt.Fprintf(os.Stderr, "%s failed: %v\n", name, err)
	os.Exit(1)
}
