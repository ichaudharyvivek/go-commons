package errorsx

import (
	"errors"
	"fmt"
)

// base represents the base error type.
type base struct {
	err   error
	stack *stacktrace
}

// New creates a new error with the given message.
func New(msg string) error {
	return &base{
		err:   errors.New(msg),
		stack: ptr(newStackTrace()),
	}
}

// Newf creates a new error with a formatted message.
func Newf(format string, args ...any) error {
	return &base{
		err:   fmt.Errorf(format, args...),
		stack: ptr(newStackTrace()),
	}
}

// Wrap an error with a message.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}

	return &base{
		err:   fmt.Errorf("%s: %w", msg, err),
		stack: ptr(newStackTrace()),
	}
}

// Wrap an error with a formatted message.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return &base{
		err:   fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err),
		stack: ptr(newStackTrace()),
	}
}

// Implements the error interface. Returns the error message.
func (e *base) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *base) Unwrap() error {
	return errors.Unwrap(e.err)
}

// Format the error with stack traces.
func (e *base) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			if trace, ok := GetStack(e); ok {
				fmt.Fprintf(s, "%s\n%s", e.Error(), trace)
				return
			}
		}
		fallthrough
	case 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// `Is` reports whether any error in the chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// `As` finds the first error in the chain that matches target and sets target to that value.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// `Unwrap` returns the result of calling the Unwrap method on the error, or nil.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// `Join` returns an error that wraps the given errors.
func Join(errs ...error) error {
	return errors.Join(errs...)
}
