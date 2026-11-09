package errorsx

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

type stacktrace []uintptr

// Captures the current call stack.
//
//	Frame 0 = runtime.Callers   		-> internal Go runtime (skipped)
//	Frame 1 = newStackTrace		        -> internal errorsx package (skipped)
//	Frame 2 = errorsx.New/Wrap/Wrapf	-> first frame we record
//	Frame 3 = caller 			-> user code (e.g. repo.GetByID)
//	Frame n = ... 				-> user code (further stacktrace)
func newStackTrace() stacktrace {
	const stackDepth = 32
	pc := make([]uintptr, stackDepth)
	n := runtime.Callers(2, pc)
	return stacktrace(pc[:n:n])
}

// ptr returns a pointer to the given stacktrace.
func ptr(s stacktrace) *stacktrace {
	return &s
}

// String formats the stack traces, skipping runtime frames.
func (s stacktrace) String() string {
	if len(s) == 0 {
		return ""
	}

	var buf strings.Builder
	frames := runtime.CallersFrames(s)
	for {
		frame, more := frames.Next()
		if strings.HasPrefix(frame.Function, "runtime.") {
			if !more {
				break
			}
			continue
		}

		buf.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}

	return buf.String()
}

// GetStack returns the stack trace for the error.
func GetStack(err error) (string, bool) {
	if err == nil {
		return "", false
	}

	var last *stacktrace
	for e := err; e != nil; e = errors.Unwrap(e) {
		if b, ok := e.(*base); ok && b.stack != nil {
			last = b.stack
		}
	}
	if last == nil {
		return "", false
	}

	s := last.String()
	if s == "" {
		return "", false
	}

	return s, true
}
