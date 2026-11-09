package httpx

import (
	"encoding/json"
	"net/http"
)

type Builder struct {
	statusCode int
	resp       Response
}

func Status(code int) *Builder {
	b := &Builder{statusCode: code}
	b.resp.Success = code >= 200 && code < 300
	return b
}

func (b *Builder) Message(msg string) *Builder {
	b.resp.Message = msg
	return b
}

func (b *Builder) Data(data any) *Builder {
	b.resp.Data = data
	return b
}

func (b *Builder) Error(err string) *Builder {
	b.resp.Error = err
	b.resp.Success = false
	return b
}

func (b *Builder) Errors(errs []string) *Builder {
	b.resp.Errors = errs
	b.resp.Success = false
	return b
}

func (b *Builder) Meta(meta *Meta) *Builder {
	b.resp.Meta = meta
	return b
}

func (b *Builder) JSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(b.statusCode)
	json.NewEncoder(w).Encode(b.resp)
}
