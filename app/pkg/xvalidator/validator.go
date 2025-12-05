package xvalidator

import (
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/nyaruka/phonenumbers"

	"github.com/Praisindo/pkg-library/app/pkg/utils/phonenumber"
	"github.com/Praisindo/pkg-library/app/pkg/utils/pointer"
)

// Validator is a struct that contains a pointer to a validator.Validate instance.
type Validator struct {
	validate *validator.Validate
}

// NewValidator is a function that initializes a new Validator instance.
// It registers a tag name function that returns the "name" tag of a struct field.
// It logs that the validator has been initialized and returns the new Validator instance.
func NewValidator() (*Validator, error) {
	validate := validator.New()
	if err := _regisValidateMYNumber(validate); err != nil {
		return nil, err
	}
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		return field.Tag.Get("name")
	})

	validate.RegisterValidation("int_list", func(fl validator.FieldLevel) bool {
		field := fl.Field()

		// Check if the field is of type []int
		if field.Kind() != reflect.Slice || field.Type().Elem().Kind() != reflect.Int {
			return false
		}

		// For example, ensure all elements are positive
		for i := 0; i < field.Len(); i++ {
			if field.Index(i).Int() < 0 {
				return false
			}
		}

		return true
	})

	validate.RegisterValidation("date_format", func(fl validator.FieldLevel) bool {
		dateStr := fl.Field().String()

		// Parse Date String
		_, err := time.Parse("2006-01-02", dateStr)
		return err == nil
	})

	validate.RegisterValidation("ratio_value", func(fl validator.FieldLevel) bool {
		valueFloat := fl.Field().Float()
		valueStr := strconv.FormatFloat(valueFloat, 'f', -1, 64) // Konversi float64 ke string
		ratioRegex := regexp.MustCompile(`^[0-9]+(\.[0-9][0-9]?)?$`)
		return ratioRegex.MatchString(valueStr)
	})
	validate.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		phoneNumber := fl.Field().String()

		phoneString, err := phonenumbers.Parse(phoneNumber, "MY")
		if err != nil {
			return false
		}
		return phonenumbers.IsValidNumber(phoneString)
	})

	validate.RegisterValidation("email_if_type", func(fl validator.FieldLevel) bool {
		typeField := fl.Parent().FieldByName("Type").String()
		emailField := fl.Field().String()
		if typeField == "email" {
			if err := validator.New().Var(emailField, "email"); err != nil {
				return false
			}
		}
		return true
	})

	validate.RegisterValidation("phone_if_type", func(fl validator.FieldLevel) bool {
		typeField := fl.Parent().FieldByName("Type").String()
		phoneNumber := fl.Field().String()

		if typeField == "phone_number" {
			phoneString, err := phonenumbers.Parse(phoneNumber, "MY")
			if err != nil {
				return false
			}
			return phonenumbers.IsValidNumber(phoneString)
		}
		return true
	})
	slog.Info("validator initialized")
	return &Validator{validate: validate}, nil
}

// _regisValidateMYNumber is a private function that registers a custom validation rule for Malaysian phone numbers.
func _regisValidateMYNumber(validate *validator.Validate) error {
	if err := validate.RegisterValidation(strings.ToLower(string(phonenumber.RegionCodeMalaysia))+"-phone-number", _validatePhoneNumber(), true); err != nil {
		slog.Error("failed to register custom validation", "error", err.Error())
		return err
	}
	return nil
}

// _validatePhoneNumber is a function that returns a function which validates a phone number.
// The returned function takes a validator.FieldLevel instance as an argument.
func _validatePhoneNumber() func(fl validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		if fl.Field().String() == reflect.ValueOf(pointer.String("")).String() {
			return true
		}
		parse, err := phonenumber.NewPhoneNumber(fl.Field().String(), phonenumber.RegionCodeMalaysia)
		if err != nil {
			return false
		}
		if !parse.IsValid() {
			return false
		}
		return true
	}
}

// Struct is a method of the Validator struct that validates a struct.
// It returns a slice of strings containing the validation errors.
// If there are no validation errors, it returns nil.
func (v *Validator) Struct(s interface{}) map[string]string {
	err := v.validate.Struct(s)
	if err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// Var is a method of the Validator struct that validates a single variable.
// It returns a slice of strings containing the validation errors.
// If there are no validation errors, it returns nil.
func (v *Validator) Var(field interface{}, tag string) map[string]string {
	err := v.validate.Var(field, tag)
	if err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// formatValidationError is a method of the Validator struct that formats validation errors.
// It returns a slice of strings containing the formatted validation errors.
func (v *Validator) formatValidationError(err error) map[string]string {
	errors := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		switch err.Tag() {
		case "required":
			errors[err.Field()] = fmt.Sprintf("%s is required", err.Field())
		case "email":
			errors[err.Field()] = fmt.Sprintf("%s is not a valid email", err.Field())
		case "min":
			errors[err.Field()] = fmt.Sprintf("%s must be at least %s", err.Field(), err.Param())
		case "max":
			errors[err.Field()] = fmt.Sprintf("%s must be at most %s", err.Field(), err.Param())
		case "len":
			errors[err.Field()] = fmt.Sprintf("%s must be %s characters long", err.Field(), err.Param())
		case "gte":
			errors[err.Field()] = fmt.Sprintf("%s must be greater than or equal to %s", err.Field(), err.Param())
		case "gt":
			errors[err.Field()] = fmt.Sprintf("%s must be greater than %s", err.Field(), err.Param())
		case "lte":
			errors[err.Field()] = fmt.Sprintf("%s must be less than or equal to %s", err.Field(), err.Param())
		case "lt":
			errors[err.Field()] = fmt.Sprintf("%s must be less than %s", err.Field(), err.Param())
		case "numeric":
			errors[err.Field()] = fmt.Sprintf("%s must be numeric", err.Field())
		case "number":
			errors[err.Field()] = fmt.Sprintf("%s must be a number", err.Field())
		case "phone":
			errors[err.Field()] = fmt.Sprintf("%s invalid phone number", err.Field())
		default:
			errors[err.Field()] = fmt.Sprintf("%s is not valid", err.Field())
		}
	}
	return errors
}
