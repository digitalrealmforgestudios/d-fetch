package grpc

type ContextKey int8

const (
	ContextRequestId ContextKey = iota + 1
)
