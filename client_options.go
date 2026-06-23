package dfetch

type SetClientOptionsFn func(*clientOptions)

type clientOptions struct {
	namespace    string
	logDump      bool
	disableHTTP2 bool
}

func defaultClientOptions() clientOptions {
	return clientOptions{
		namespace: "d-fetch",
	}
}

// Namespace changes the logger namespace used by a client.
func Namespace(namespace string) SetClientOptionsFn {
	return func(o *clientOptions) {
		o.namespace = namespace
	}
}

// LogDump enables or disables HTTP request and response dumps.
func LogDump(enable bool) SetClientOptionsFn {
	return func(o *clientOptions) {
		o.logDump = enable
	}
}

// DisableHTTP2 disables Go's automatic HTTP/2 transport upgrade for the client.
func DisableHTTP2() SetClientOptionsFn {
	return func(o *clientOptions) {
		o.disableHTTP2 = true
	}
}

func evaluateClientOptions(fns []SetClientOptionsFn) clientOptions {
	options := defaultClientOptions()
	for _, fn := range fns {
		if fn != nil {
			fn(&options)
		}
	}
	return options
}
