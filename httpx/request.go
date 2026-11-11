package httpx

import (
	"encoding/json"
	"net/http"
)

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
