package body

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	contentTypeJSON = "application/json"
	contentTypeForm = "application/x-www-form-urlencoded"
	methodGet       = "GET"
)

func Compose(method, contentType string, value interface{}) ([]byte, error) {
	if method == methodGet || value == nil {
		return nil, nil
	}
	if raw, ok := value.([]byte); ok {
		return raw, nil
	}

	switch contentType {
	case contentTypeJSON:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("d-fetch: Failed to compose request body. ContentType = %s, Error = %w", contentType, err)
		}
		return encoded, nil
	case contentTypeForm:
		form, ok := value.(url.Values)
		if !ok {
			return nil, fmt.Errorf("d-fetch: Unable to compose URL-Encoded Form, body is not url.Values type. Type = %T", value)
		}
		return []byte(form.Encode()), nil
	default:
		return nil, nil
	}
}

func Unsupported(method, contentType string, value interface{}, composed []byte) bool {
	return method != methodGet && value != nil && len(composed) == 0 && contentType != ""
}
