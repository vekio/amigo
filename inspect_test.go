package amigo

import (
	"context"
	"testing"
)

func TestInvalidInputDeclarationsPanicDuringRegistration(t *testing.T) {
	tests := []struct {
		name    string
		message string
		build   func(*API)
	}{
		{
			name:    "non struct input",
			message: "input must be a struct",
			build: func(api *API) {
				api.GET("/", func(_ context.Context, _ int) (struct{}, error) { return struct{}{}, nil })
			},
		},
		{
			name:    "multiple sources",
			message: "cannot have multiple parameter sources",
			build: func(api *API) {
				type input struct {
					Value string `query:"value" header:"X-Value"`
				}
				api.GET("/", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
			},
		},
		{
			name:    "empty source tag",
			message: `query field "Value" has an empty tag`,
			build: func(api *API) {
				type input struct {
					Value string `query:""`
				}
				api.GET("/", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
			},
		},
		{
			name:    "unexported parameter",
			message: `parameter field "value" must be exported`,
			build: func(api *API) {
				type input struct {
					value string `query:"value"`
				}
				api.GET("/", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
			},
		},
		{
			name:    "default outside query",
			message: "cannot have a default",
			build: func(api *API) {
				type input struct {
					Value string `header:"X-Value" default:"x"`
				}
				api.GET("/", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
			},
		},
		{
			name:    "validation without source",
			message: "has validation rules but is not a request parameter",
			build: func(api *API) {
				type input struct {
					Value string `validate:"rule"`
				}
				api.GET("/", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
			},
		},
		{
			name:    "empty validation tag",
			message: "has an empty validate tag",
			build: func(api *API) {
				type input struct {
					Value string `query:"value" validate:""`
				}
				api.GET("/", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
			},
		},
		{
			name:    "body is also parameter",
			message: "cannot be both body and query parameter",
			build: func(api *API) {
				type input struct {
					Body struct{} `query:"body"`
				}
				api.GET("/", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requirePanicContains(t, test.message, func() {
				test.build(New())
			})
		})
	}
}

func TestRecursiveBodyMetadataTerminates(t *testing.T) {
	type node struct {
		Value string `json:"value" validate:"required"`
		Next  *node  `json:"next"`
	}
	type input struct {
		Body node
	}

	api := New()
	api.Validator("required", func(string) error { return nil })
	api.POST("/nodes", func(_ context.Context, _ input) (struct{}, error) { return struct{}{}, nil })
	operations := api.Operations()
	if len(operations) != 1 || len(operations[0].Input.Validations) != 1 {
		t.Fatalf("operations = %#v", operations)
	}
}
