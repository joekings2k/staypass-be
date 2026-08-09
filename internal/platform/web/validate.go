package web

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

func DecodeAndValidate(w http.ResponseWriter, r *http.Request, dst any) error {
	if err := Decode(w, r, dst); err != nil {
		return err
	}
	return validate.Struct(dst)
}

func InvalidRequest(w http.ResponseWriter, err error) {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		BadRequest(w, "invalid request body")
		return
	}

	fields := make(map[string]string, len(ve))
	for _, fe := range ve {
		fields[fe.Field()] = fieldMessage(fe)
	}
	JSON(w, http.StatusUnprocessableEntity, errorBody{Error: errorDetail{
		Code:    "validation",
		Message: "one or more fields are invalid",
		Fields:  fields,
	}})
}

func fieldMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	default:
		return "is invalid"
	}
}
