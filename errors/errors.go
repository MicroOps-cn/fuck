package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm/logger"

	w "github.com/MicroOps-cn/fuck/wrapper"
)

type Error interface {
	Code() string
	StatusCode() int
	error
}

// Is reports whether any error in err's chain matches target.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
func Is(err, target error) bool { return stderrors.Is(err, target) }

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// As will panic if target is not a non-nil pointer to either a type that implements
// error, or to any interface type. As returns false if err is nil.
func As(err error, target interface{}) bool { return stderrors.As(err, target) }

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

var Join = stderrors.Join

type errorWrapper struct {
	code   string
	status int
	err    error
}

func (s *errorWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Error())
}

func (s *errorWrapper) String() string {
	return s.Error()
}

func (s *errorWrapper) Code() string {
	return s.code
}
func (s *errorWrapper) Unwrap() error {
	return s.err
}

func (s *errorWrapper) StatusCode() int {
	return s.status
}

func (s *errorWrapper) Error() string {
	return s.err.Error()
}

func (s *errorWrapper) Format(state fmt.State, verb rune) {
	switch verb {
	case 'v':
		if state.Flag('+') {
			if f, ok := s.err.(fmt.Formatter); ok {
				f.Format(state, verb)
			}
			return
		}
		fallthrough
	case 's':
		io.WriteString(state, s.Error())
	case 'q':
		fmt.Fprintf(state, "%q", s.Error())
	}
}

func WithMessage(err error, msg string) error {
	var ew *errorWrapper
	if errors.As(err, &ew) {
		return &errorWrapper{
			code:   ew.code,
			status: ew.status,
			err:    errors.WithMessage(ew.err, msg),
		}
	}
	var e Error
	if errors.As(err, &e) {
		return &errorWrapper{
			code:   e.Code(),
			status: e.StatusCode(),
			err:    errors.WithMessage(err, msg),
		}
	}
	if errors.Is(err, logger.ErrRecordNotFound) {
		return &errorWrapper{
			code:   "404",
			status: http.StatusNotFound,
			err:    errors.WithMessage(err, msg),
		}
	}
	return errors.WithMessage(err, msg)
}

func With(status int, err error, msg string, code ...string) Error {
	var c string
	if len(code) == 0 {
		c = strconv.Itoa(status)
	} else {
		c = code[0]
	}
	if msg != "" {
		err = errors.WithMessage(err, msg)
	}
	return &errorWrapper{
		code:   c,
		status: status,
		err:    err,
	}
}

func New(msg string) error {
	return NewError(500, msg)
}
func NewError(status int, msg string, code ...string) Error {
	var c string
	if len(code) == 0 {
		c = strconv.Itoa(status)
	} else {
		c = code[0]
	}
	return &errorWrapper{
		code:   c,
		status: status,
		err:    errors.New(msg),
	}
}

var NotFoundError = NewError(404, "record not found")

type notFoundError struct {
	name string
}

func (e notFoundError) Error() string {
	return "record not found: " + e.name
}

func NewNotFoundError(name string) error {
	return &notFoundError{name: name}
}

func IsNotFount(err error) bool {
	if errors.Is(err, NotFoundError) || errors.Is(err, logger.ErrRecordNotFound) {
		return true
	}
	var ne *notFoundError
	if errors.As(err, &ne) {
		return true
	}
	var e *errorWrapper
	if errors.As(err, &e) {
		if e.StatusCode() == 404 {
			return true
		}
		return IsNotFount(e.err)
	}
	return false
}

func NewErrors(status int, prefix string, code ...string) *Errors {
	var c string
	if len(code) <= 0 {
		c = strconv.Itoa(status)
	} else {
		c = code[0]
	}
	return &Errors{
		code:   c,
		status: status,
		errs:   []error{},
		prefix: prefix,
	}
}

type Errors struct {
	errs   []error
	code   string
	status int
	prefix string
}

func (m *Errors) Code() string {
	return m.code
}

func (m *Errors) StatusCode() int {
	return m.status
}

func (m *Errors) Error() string {
	if len(m.errs) > 0 {
		if len(m.errs) == 1 {
			return m.errs[0].Error()
		}
		var errs []string
		for _, err := range m.errs {
			errs = append(errs, err.Error())
		}
		return m.prefix + strings.Join(errs, ",")
	}
	return ""
}

func (m *Errors) HasError() bool {
	return len(w.Filter(m.errs, func(err error) bool {
		return err != nil
	})) > 0
}

func (m *Errors) Append(err error) {
	if err != nil {
		m.errs = append(m.errs, err)
	}
}

var _ Error = (*Errors)(nil)
