package dfetch

import (
	"errors"
	"net/http"
	"net/url"
)

type SetRequestOptionFn func(*requestOptions)

type PreRequestFn func(req *http.Request, reqBody []byte)

type requestOptions struct {
	canonicalHeader bool
	headers         map[string]string
	query           url.Values
	body            interface{}
	timeoutMS       int
	preRequest      PreRequestFn
}

func defaultRequestOptions() requestOptions {
	return requestOptions{
		canonicalHeader: true,
		headers:         make(map[string]string),
		query:           make(url.Values),
		timeoutMS:       10000,
	}
}

func AddHeader(args ...string) SetRequestOptionFn {
	return func(o *requestOptions) {
		requirePairs("AddHeader", args)
		for i := 0; i < len(args); i += 2 {
			o.headers[args[i]] = args[i+1]
		}
	}
}

func AddQuery(args ...string) SetRequestOptionFn {
	return func(o *requestOptions) {
		requirePairs("AddQuery", args)
		for i := 0; i < len(args); i += 2 {
			o.query.Add(args[i], args[i+1])
		}
	}
}

func SetBody(body interface{}) SetRequestOptionFn {
	return func(o *requestOptions) {
		if body != nil {
			o.body = body
		}
	}
}

func SetJsonBody(body interface{}) SetRequestOptionFn {
	return func(o *requestOptions) {
		if body != nil {
			o.headers[HeaderContentType] = MimeTypeJson
			o.body = body
		}
	}
}

func SetUrlEncodedFormBody(body url.Values) SetRequestOptionFn {
	return func(o *requestOptions) {
		if body != nil {
			o.headers[HeaderContentType] = MimeTypeUrlEncodedForm
			o.body = body
		}
	}
}

func Timeout(ms int) SetRequestOptionFn {
	return func(o *requestOptions) {
		o.timeoutMS = ms
	}
}

func PreRequest(fn PreRequestFn) SetRequestOptionFn {
	return func(o *requestOptions) {
		o.preRequest = fn
	}
}

func DisableCanonicalHeader() SetRequestOptionFn {
	return func(o *requestOptions) {
		o.canonicalHeader = false
	}
}

func evaluateRequestOptions(fns []SetRequestOptionFn) requestOptions {
	options := defaultRequestOptions()
	for _, fn := range fns {
		if fn != nil {
			fn(&options)
		}
	}
	return options
}

func requirePairs(name string, args []string) {
	if len(args) == 0 || len(args)%2 == 1 {
		panic(errors.New("d-fetch: Invalid " + name + "() args count must >= 2 and even"))
	}
}
