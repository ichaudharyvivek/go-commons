package observability

import (
	"crypto/subtle"
	"net/http"
	"net/http/pprof"
)

// pprofProfiles are the named runtime profiles beyond the five pprof.*
// handler funcs (index/cmdline/profile/symbol/trace).
var pprofProfiles = []string{"goroutine", "heap", "threadcreate", "block", "mutex", "allocs"}

// ProfilingHandler returns an http.Handler serving the standard net/http/pprof
// endpoints behind HTTP basic auth, using cfg's credentials.
//
// Meaningful only when Profiling.Enabled. Per production guidance, pprof output can leak
// sensitive runtime information and MUST NOT be exposed publicly — mount this
// on an internal-only port/route, never on a public one.
func ProfilingHandler(cfg ProfilingConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	for _, name := range pprofProfiles {
		mux.Handle("/debug/pprof/"+name, pprof.Handler(name))
	}

	return basicAuth(cfg.BasicAuthUser, cfg.BasicAuthPass, mux)
}

func basicAuth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok := r.BasicAuth()
		validUser := subtle.ConstantTimeCompare([]byte(gotUser), []byte(user)) == 1
		validPass := subtle.ConstantTimeCompare([]byte(gotPass), []byte(pass)) == 1
		if !ok || !validUser || !validPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
