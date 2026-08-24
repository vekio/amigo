package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
	"uuid"

	"github.com/vekio/amigo"
)

type GetUserInput struct {
	ID            uuid.UUID `path:"id"`
	Authorization *string   `header:"Authorization"`
	Session       *string   `cookie:"session"`
}

type GetUserOutput struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Auth    *string   `json:"authorization"`
	Session *string   `json:"session"`
}

type UserNotFoundError struct{}

func (UserNotFoundError) Error() string {
	return "user not found"
}

func (UserNotFoundError) StatusCode() int {
	return http.StatusNotFound
}

func GetUser(_ context.Context, input GetUserInput) (GetUserOutput, error) {
	if input.ID == uuid.Nil() {
		return GetUserOutput{}, UserNotFoundError{}
	}

	return GetUserOutput{
		ID:      input.ID,
		Name:    "Ada",
		Auth:    input.Authorization,
		Session: input.Session,
	}, nil
}

type ListUsersInput struct {
	Page  int        `query:"page" default:"1"`
	Limit int        `query:"limit" default:"20"`
	Tags  []string   `query:"tags"`
	Since *time.Time `query:"since"`
}

type ListUsersOutput struct {
	Page   int        `json:"page"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
	Tags   []string   `json:"tags"`
	Since  *time.Time `json:"since"`
}

func ListUsers(_ context.Context, input ListUsersInput) (ListUsersOutput, error) {
	return ListUsersOutput{
		Page:   input.Page,
		Limit:  input.Limit,
		Offset: (input.Page - 1) * input.Limit,
		Tags:   input.Tags,
		Since:  input.Since,
	}, nil
}

type CreateUserBody struct {
	Name    string            `json:"name"`
	Email   string            `json:"email"`
	Age     int               `json:"age"`
	Address CreateUserAddress `json:"address"`
}

type CreateUserAddress struct {
	City string `json:"city"`
}

type CreateUserInput struct {
	Authorization *string `header:"Authorization"`
	Body          CreateUserBody
}

type CreateUserOutput struct {
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	Age           int     `json:"age"`
	City          string  `json:"city"`
	Authorization *string `json:"authorization"`
}

func CreateUser(_ context.Context, input CreateUserInput) (CreateUserOutput, error) {
	validation := &amigo.ValidationError{}
	if input.Body.Email == "" {
		validation.Errors = append(validation.Errors, amigo.FieldError{
			Location: "body.email",
			Message:  "value is required",
		})
	}
	if input.Body.Age < 0 {
		validation.Errors = append(validation.Errors, amigo.FieldError{
			Location: "body.age",
			Message:  "must be greater than or equal to 0",
		})
	}
	if input.Body.Address.City == "" {
		validation.Errors = append(validation.Errors, amigo.FieldError{
			Location: "body.address.city",
			Message:  "value is required",
		})
	}
	if len(validation.Errors) > 0 {
		return CreateUserOutput{}, validation
	}

	return CreateUserOutput{
		Name:          input.Body.Name,
		Email:         input.Body.Email,
		Age:           input.Body.Age,
		City:          input.Body.Address.City,
		Authorization: input.Authorization,
	}, nil
}

type ErrorDemoInput struct {
	Kind string `path:"kind"`
}

type ErrorDemoOutput struct {
	Message string `json:"message"`
}

type ServiceUnavailableError struct{}

func (ServiceUnavailableError) Error() string {
	return "service is temporarily unavailable"
}

func (ServiceUnavailableError) StatusCode() int {
	return http.StatusServiceUnavailable
}

var errDatabaseUnavailable = errors.New("database connection failed")

func ErrorDemo(_ context.Context, input ErrorDemoInput) (ErrorDemoOutput, error) {
	switch input.Kind {
	case "problem":
		problem := amigo.Conflict("a user with that email already exists")
		problem.Type = "https://example.com/problems/user-conflict"
		return ErrorDemoOutput{}, problem
	case "status":
		return ErrorDemoOutput{}, ServiceUnavailableError{}
	case "validation":
		return ErrorDemoOutput{}, &amigo.ValidationError{
			Errors: []amigo.FieldError{
				{Location: "query.page", Message: "must be greater than 0"},
			},
		}
	case "wrapped":
		return ErrorDemoOutput{}, amigo.WrapProblem(
			errDatabaseUnavailable,
			amigo.InternalServerError("internal server error"),
		)
	case "private":
		return ErrorDemoOutput{}, errors.New("private repository error")
	default:
		return ErrorDemoOutput{Message: "no error"}, nil
	}
}

type NoInput struct{}

type EncodingErrorOutput struct {
	Unsupported func() `json:"unsupported"`
}

func EncodingError(_ context.Context, _ NoInput) (EncodingErrorOutput, error) {
	return EncodingErrorOutput{Unsupported: func() {}}, nil
}

func main() {
	app := amigo.New()
	app.SetErrorHandler(exampleErrorHandler)
	api := amigo.NewRouter(
		amigo.Prefix("/api"),
		amigo.Tags("api"),
	)
	users := amigo.NewRouter(
		amigo.Prefix("/users"),
		amigo.Tags("users"),
	)
	errorRoutes := amigo.NewRouter(
		amigo.Prefix("/errors"),
		amigo.Tags("errors"),
	)

	api.Use(loggingMiddleware)
	users.GET("", ListUsers, amigo.Tags("list"))
	users.POST("", CreateUser, amigo.Tags("create"))
	users.GET("/{id}", GetUser, amigo.Tags("read"))
	errorRoutes.GET("/encoding", EncodingError, amigo.Tags("encoding"))
	errorRoutes.GET("/{kind}", ErrorDemo, amigo.Tags("demo"))
	api.Include(users)
	api.Include(errorRoutes)
	app.Include(api)

	log.Println("listening on http://localhost:8080")
	log.Println(`try: curl "http://localhost:8080/api/users?page=2&limit=10&tags=go&tags=http&since=2026-08-24T10:30:00Z"`)
	log.Println(`validation: curl -X POST -H "Content-Type: application/json" -d '{"name":"Grace","age":-1,"address":{}}' http://localhost:8080/api/users`)
	log.Println(`problem: curl http://localhost:8080/api/errors/problem`)
	log.Println(`status error: curl http://localhost:8080/api/errors/status`)
	log.Println(`wrapped problem: curl http://localhost:8080/api/errors/wrapped`)
	log.Println(`private error: curl http://localhost:8080/api/errors/private`)
	log.Println(`encoding error: curl http://localhost:8080/api/errors/encoding`)
	log.Println(`try: curl -H "Authorization: Bearer demo" --cookie "session=abc" http://localhost:8080/api/users/01941f29-7c00-7d00-a5c9-345f08c39fbd`)
	log.Fatal(app.Run(":8080"))
}

func exampleErrorHandler(
	w http.ResponseWriter,
	req *http.Request,
	phase amigo.ErrorPhase,
	err error,
) {
	log.Printf("request error: phase=%s method=%s path=%s", phase, req.Method, req.URL.Path)
	if errors.Is(err, errDatabaseUnavailable) {
		log.Printf("wrapped cause preserved: %v", errDatabaseUnavailable)
	}
	amigo.DefaultErrorHandler(w, req, phase, err)
}

func loggingMiddleware(w http.ResponseWriter, req *http.Request, next http.Handler) {
	started := time.Now()
	next.ServeHTTP(w, req)
	log.Printf("%s %s %s", req.Method, req.URL.Path, time.Since(started))
}
