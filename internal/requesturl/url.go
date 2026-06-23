package requesturl

import (
	"net/url"
	"strings"
)

func Join(baseURL, endpointPath string, query url.Values) string {
	var b strings.Builder
	b.WriteString(baseURL)
	b.WriteString(endpointPath)
	if len(query) > 0 {
		b.WriteString("?")
		b.WriteString(query.Encode())
	}
	return b.String()
}
