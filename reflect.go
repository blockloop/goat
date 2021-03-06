package boar

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/blockloop/boar/bind"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/go-playground/validator.v9"
)

const (
	queryField     = "Query"
	urlParamsField = "URLParams"
	bodyField      = "Body"
)

var (
	// MultiPartFormMaxMemory says how much memory to send to (*http.Request).ParseMultipartForm
	// Default is 2MB
	MultiPartFormMaxMemory = int64(1 << 20) // 2MB

	errNotAStruct    = errors.New("not a struct")
	errNotSettable   = errors.New("not settable")
	errNoContentType = errors.New("content-type header was not set on the request")

	contentTypeJSON          = "application/json"
	contentTypeFormEncoded   = "application/x-www-form-urlencoded"
	contentTypeMultipartForm = "multipart/form-data"

	validateImpl = validator.New()
)

func checkField(field reflect.Value) (bool, error) {
	if !field.IsValid() {
		return false, nil
	}
	if field.Kind() != reflect.Struct {
		return false, errNotAStruct
	}
	if !field.CanSet() {
		return false, errNotSettable
	}
	return true, nil
}

func setQuery(handler reflect.Value, qs url.Values) error {
	field := handler.FieldByName(queryField)
	ok, err := checkField(field)
	if !ok {
		if err == nil {
			return nil
		}
		return &badFieldError{
			field:   queryField,
			handler: handler,
			err:     err,
		}
	}
	if err := bind.QueryValue(field, qs); err != nil {
		return NewValidationError(queryField, err)
	}
	return validate(queryField, field.Addr().Interface())
}

func setURLParams(handler reflect.Value, params httprouter.Params) error {
	field := handler.FieldByName(urlParamsField)
	ok, err := checkField(field)
	if !ok {
		if err == nil {
			return nil
		}
		return &badFieldError{
			field:   urlParamsField,
			handler: handler,
			err:     err,
		}
	}
	if err := bind.ParamsValue(field, params); err != nil {
		if tme, ok := err.(*bind.TypeMismatchError); ok {
			return NewValidationError(urlParamsField, tme)
		}
		return err
	}
	return validate(urlParamsField, field.Addr().Interface())
}

func setBody(handler reflect.Value, c Context) error {
	field := handler.FieldByName(bodyField)
	ok, err := checkField(field)
	if !ok {
		if err == nil {
			return nil
		}
		return &badFieldError{
			field:   bodyField,
			handler: handler,
			err:     err,
		}
	}
	binder, err := getBinder(c)
	if err != nil {
		return NewHTTPError(http.StatusBadRequest, err)
	}

	if err := binder(field.Addr().Interface()); err != nil {
		return NewValidationError(bodyField, err)
	}
	return validate(bodyField, field.Addr().Interface())
}

type binderFunc func(interface{}) error

func getBinder(c Context) (binderFunc, error) {
	ct := c.Request().Header.Get("content-type")
	switch ct {
	case "":
		return nil, errNoContentType
	case contentTypeJSON:
		return c.ReadJSON, nil
	case contentTypeFormEncoded:
		return c.ReadForm, c.Request().ParseForm()
	default:
		if strings.HasPrefix(ct, contentTypeMultipartForm) {
			return c.ReadForm, c.Request().ParseMultipartForm(MultiPartFormMaxMemory)
		}
		return nil, fmt.Errorf("unknown content type: %q", ct)
	}
}

func validate(fieldName string, v interface{}) error {
	if err := validateImpl.Struct(v); err != nil {
		return NewValidationErrors(fieldName, []error{err})
	}
	return nil
}

type badFieldError struct {
	field   string
	handler reflect.Value
	err     error
}

func (b badFieldError) Error() string {
	return fmt.Sprintf("%s field of %s is %s", b.field, b.handler.Type().Name(), b.err)
}
