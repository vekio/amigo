package amigo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func emptyOperationEndpoint(context.Context, struct{}) (struct{}, error) {
	return struct{}{}, nil
}

func TestDefaultOperationID(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{method: "GET", path: "/", want: "get-root"},
		{method: "POST", path: "/projects/{project_id}/links", want: "post-projects-project-id-links"},
		{method: "GET", path: "/files/{path...}", want: "get-files-path"},
		{method: "", path: "/health", want: "operation-health"},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			if got := defaultOperationID(test.method, test.path); got != test.want {
				t.Errorf("defaultOperationID() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAPIRecordsGeneratedOperationID(t *testing.T) {
	api := New()
	api.GET("/projects/{project_id}/links", func(context.Context, struct {
		ProjectID string `path:"project_id" json:"-"`
	}) (struct{}, error) {
		return struct{}{}, nil
	})

	if got := api.operations[0].operationID; got != "get-projects-project-id-links" {
		t.Errorf("operation ID = %q, want get-projects-project-id-links", got)
	}
}

func TestAPIRejectsDuplicateGeneratedOperationIDBeforeRegisteringRoute(t *testing.T) {
	api := New()
	api.GET("/foo-bar", emptyOperationEndpoint)

	assertPanics(t, func() {
		api.GET("/foo/bar", emptyOperationEndpoint)
	})

	if len(api.operations) != 1 {
		t.Errorf("operations = %d, want 1", len(api.operations))
	}
	response := httptest.NewRecorder()
	api.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/foo/bar", nil))
	if response.Code != http.StatusNotFound {
		t.Errorf("rejected route status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestWithOperationIDResolvesGeneratedIDCollision(t *testing.T) {
	api := New()
	api.GET("/foo-bar", emptyOperationEndpoint)
	api.GET("/foo/bar", emptyOperationEndpoint, WithOperationID("getNestedFooBar"))

	if len(api.operations) != 2 || api.operations[1].operationID != "getNestedFooBar" {
		t.Errorf("operations = %#v", api.operations)
	}
}

type recordedOperationInput struct {
	ID   string `path:"id" json:"-"`
	Name string `json:"name" validate:"required"`
}

type recordedOperationOutput struct {
	Location string `header:"Location" json:"-"`
	ID       string `json:"id"`
}

func TestAPIRecordsTypedOperationsInRegistrationOrder(t *testing.T) {
	conflict := errors.New("link already exists")
	api := New()
	links := api.Group("/v1/links")
	links.POST("/{id}", func(context.Context, recordedOperationInput) (recordedOperationOutput, error) {
		return recordedOperationOutput{}, nil
	},
		WithStatus(http.StatusCreated),
		WithOperationID("createLink"),
		WithSummary("Create a shortened link"),
		WithDescription("Creates a link for a target URL."),
		WithTags("Links"),
		WithErrorMapping(conflict, http.StatusConflict, "link already exists"),
	)
	api.GET("/health", func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, WithOperationID("health"))

	if len(api.operations) != 2 {
		t.Fatalf("operations = %d, want 2", len(api.operations))
	}

	operation := api.operations[0]
	if operation.method != http.MethodPost || operation.path != "/v1/links/{id}" {
		t.Errorf("operation route = %s %s", operation.method, operation.path)
	}
	if operation.status != http.StatusCreated || operation.operationID != "createLink" {
		t.Errorf("status = %d, operation ID = %q", operation.status, operation.operationID)
	}
	if operation.summary != "Create a shortened link" || operation.description != "Creates a link for a target URL." {
		t.Errorf("summary = %q, description = %q", operation.summary, operation.description)
	}
	if !reflect.DeepEqual(operation.tags, []string{"Links"}) {
		t.Errorf("tags = %#v", operation.tags)
	}
	if operation.inputType != reflect.TypeFor[recordedOperationInput]() {
		t.Errorf("input type = %s", operation.inputType)
	}
	if operation.outputType != reflect.TypeFor[recordedOperationOutput]() {
		t.Errorf("output type = %s", operation.outputType)
	}
	if len(operation.input.pathParameters) != 1 || len(operation.input.body.fields) != 1 {
		t.Errorf("input metadata = %#v", operation.input)
	}
	if len(operation.output.headers) != 1 || len(operation.output.body.fields) != 1 {
		t.Errorf("output metadata = %#v", operation.output)
	}
	if len(operation.errors) != 1 || !errors.Is(operation.errors[0].target, conflict) {
		t.Errorf("errors = %#v", operation.errors)
	}
	if api.operations[1].operationID != "health" {
		t.Errorf("second operation ID = %q, want health", api.operations[1].operationID)
	}
}

func TestAPIRecordsRawOperationWithoutInferredSchemas(t *testing.T) {
	api := New()
	api.RAW(http.MethodGet, "/health", func(http.ResponseWriter, *http.Request) error {
		return nil
	}, WithOperationID("rawHealth"), WithTags("Internal"))

	if len(api.operations) != 1 {
		t.Fatalf("operations = %d, want 1", len(api.operations))
	}
	operation := api.operations[0]
	if operation.inputType != nil || operation.outputType != nil {
		t.Errorf("raw types = (%v, %v), want nil", operation.inputType, operation.outputType)
	}
	if !operation.input.body.isEmpty() || len(operation.output.headers) != 0 {
		t.Errorf("raw operation has inferred metadata: %#v", operation)
	}
}
