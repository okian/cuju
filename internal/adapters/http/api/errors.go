package api

import (
	"errors"
	"fmt"
)

// Sentinel kinds for API errors.
var (
	ErrServe        = errors.New("swagger serve failed")
	ErrBadRequest   = errors.New("bad request")
	ErrBackpressure = errors.New("backpressure")
)

// Error provides operation and kind context for swagger errors.
type Error struct {
	Op   string
	Kind error
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	switch {
	case e.Op != "" && e.Kind != nil && e.Err != nil:
		return fmt.Sprintf("%s: %v: %v", e.Op, e.Kind, e.Err)
	case e.Op != "" && e.Kind != nil:
		return fmt.Sprintf("%s: %v", e.Op, e.Kind)
	case e.Op != "" && e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	case e.Kind != nil:
		return e.Kind.Error()
	case e.Err != nil:
		return e.Err.Error()
	default:
		return "unknown error"
	}
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Is(target error) bool {
	if e == nil {
		return false
	}
	if e.Kind != nil && target == e.Kind {
		return true
	}
	if e.Err != nil {
		return errors.Is(e.Err, target)
	}
	return false
}

func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Err: err}
}

func WrapKind(op string, kind, err error) error {
	if err == nil {
		return &Error{Op: op, Kind: kind}
	}
	return &Error{Op: op, Kind: kind, Err: err}
}

func NewKind(op string, kind error) error { return &Error{Op: op, Kind: kind} }
