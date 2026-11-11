package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"uuid"

	"github.com/ichaudharyvivek/go-commons/errorsx"
)

var ErrInvalidParam = errors.New("invalid parameter")

// Validatable is implemented by request DTOs.
type Validatable interface {
	// Validate checks the request DTO for invalid or missing fields.
	Validate() error
}

// DecodeAndValidate decodes a JSON request body and validates the resulting object.
func DecodeAndValidate(r *http.Request, obj Validatable) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	if err := d.Decode(obj); err != nil {
		return err
	}

	return obj.Validate()
}

// Query reads a required query param.
func Query[T any](r *http.Request, key string) (T, error) {
	return parse[T](key, r.URL.Query().Get(key))
}

// QueryOr reads a required query param. Missing/empty key returns def.
func QueryOr[T any](r *http.Request, key string, def T) (T, error) {
	if r.URL.Query().Get(key) == "" {
		return def, nil
	}
	return Query[T](r, key)
}

// Path reads and returns the URL path param.
func Path[T any](r *http.Request, key string) (T, error) {
	return parse[T](key, r.PathValue(key))
}

func parse[T any](key, raw string) (T, error) {
	var zero T
	if raw == "" {
		return zero, errorsx.Newf("%w: %q is required", ErrInvalidParam, key)
	}

	var out T
	var err error
	switch p := any(&out).(type) {
	case *string:
		*p = raw
	case *int:
		*p, err = strconv.Atoi(raw)
	case *int64:
		*p, err = strconv.ParseInt(raw, 10, 64)
	case *float64:
		*p, err = strconv.ParseFloat(raw, 64)
	case *bool:
		*p, err = strconv.ParseBool(raw)
	case *uuid.UUID:
		*p, err = uuid.Parse(raw)
	default:
		return zero, errorsx.Newf("httpx: unsupported param type %T", p)
	}
	if err != nil {
		return zero, errorsx.Newf("httpx: %w: %s: %v", ErrInvalidParam, key, err)
	}
	return out, nil
}
