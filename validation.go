package amigo

import "net/http"

// FieldError describes one invalid request value.
type FieldError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
}

// ValidationError contains all invalid request values found during binding or
// handler validation.
type ValidationError struct {
	Errors []FieldError
}

func (validation *ValidationError) Error() string {
	return "request parameters are invalid"
}

func (validation *ValidationError) StatusCode() int {
	return http.StatusBadRequest
}

func problemFromValidation(validation *ValidationError) *Problem {
	problem := BadRequest("request parameters are invalid")
	if validation != nil {
		problem.Errors = append([]FieldError(nil), validation.Errors...)
	}
	return problem
}
