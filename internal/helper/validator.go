package helper

import "github.com/go-playground/validator/v10"

type Validator struct {
	validate *validator.Validate
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return &Validator{
		validate: validate,
	}
}

// Validate validates a struct
func (v *Validator) Validate(s any) error {
	return v.validate.Struct(s)
}
