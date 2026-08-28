package amigo

import "net/http"

// problem is the RFC 9457 representation returned for endpoint errors.
type problem struct {
	Type     string       `json:"type,omitempty"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []fieldError `json:"errors,omitempty"`
}

func newProblem(status int, detail string) *problem {
	return &problem{
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
}

func (problem *problem) Error() string {
	if problem.Detail == "" {
		return problem.Title
	}
	return problem.Detail
}
