package amigo

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"uuid"
)

func TestBindInputDecodesJSON(t *testing.T) {
	type inputBody struct {
		Name string `json:"name"`
	}
	request := httptest.NewRequest(http.MethodPost, "/things", strings.NewReader(`{"name":"shorty"}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	input, err := bindInput[inputBody](request, buildInputMetadata[inputBody]("/things", newValidatorRegistry()))

	if err != nil {
		t.Fatalf("bindInput() error = %v", err)
	}
	if input.Name != "shorty" {
		t.Errorf("name = %q, want %q", input.Name, "shorty")
	}
}

func TestBindInputAllowsMissingBody(t *testing.T) {
	type inputBody struct{ Name string }
	request := httptest.NewRequest(http.MethodGet, "/things", nil)

	input, err := bindInput[inputBody](request, buildInputMetadata[inputBody]("/things", newValidatorRegistry()))

	if err != nil {
		t.Fatalf("bindInput() error = %v", err)
	}
	if input.Name != "" {
		t.Errorf("name = %q, want empty", input.Name)
	}
}

func TestBindInputRejectsInvalidJSONRequests(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		status      int
	}{
		{name: "missing content type", body: `{}`, status: http.StatusUnsupportedMediaType},
		{name: "wrong content type", body: `{}`, contentType: "text/plain", status: http.StatusUnsupportedMediaType},
		{name: "malformed JSON", body: `{"name":`, contentType: "application/json", status: http.StatusBadRequest},
		{name: "unknown member", body: `{"unknown":true}`, contentType: "application/json", status: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			type inputBody struct {
				Name string `json:"name"`
			}
			request := httptest.NewRequest(http.MethodPost, "/things", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}

			_, err := bindInput[inputBody](request, buildInputMetadata[inputBody]("/things", newValidatorRegistry()))
			problem, ok := errors.AsType[*problem](err)
			if !ok {
				t.Fatalf("error = %T, want *problem", err)
			}
			if problem.Status != test.status {
				t.Errorf("status = %d, want %d", problem.Status, test.status)
			}
		})
	}
}

func TestBindInputBindsPathParameter(t *testing.T) {
	type inputBody struct {
		ID int `path:"id" json:"-"`
	}
	request := httptest.NewRequest(http.MethodGet, "/things/42", nil)
	request.SetPathValue("id", "42")

	input, err := bindInput[inputBody](request, buildInputMetadata[inputBody]("/things/{id}", newValidatorRegistry()))

	if err != nil {
		t.Fatalf("bindInput() error = %v", err)
	}
	if input.ID != 42 {
		t.Errorf("ID = %d, want %d", input.ID, 42)
	}
}

func TestBindInputBindsQueryAndHeaderParameters(t *testing.T) {
	type input struct {
		Search    string `query:"search" json:"-"`
		Page      uint   `query:"page" json:"-"`
		RequestID string `header:"X-Request-ID" json:"-"`
		Preview   bool   `header:"X-Preview" json:"-"`
	}
	request := httptest.NewRequest(http.MethodGet, "/things?search=shorty&page=2", nil)
	request.Header.Set("x-request-id", "request-42")
	request.Header.Set("X-Preview", "true")

	got, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
	if err != nil {
		t.Fatalf("bindInput() error = %v", err)
	}
	want := input{Search: "shorty", Page: 2, RequestID: "request-42", Preview: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("input = %#v, want %#v", got, want)
	}
}

func TestBindInputBindsUUIDParameters(t *testing.T) {
	pathID := uuid.MustParse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6")
	filterID := uuid.MustParse("01934c3e-7f5d-7cc8-9f23-8b6e4f61a245")
	relatedID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	requestID := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	type input struct {
		ID        uuid.UUID   `path:"id" json:"-"`
		Filter    uuid.UUID   `query:"filter" json:"-"`
		Related   []uuid.UUID `query:"related" json:"-"`
		RequestID uuid.UUID   `header:"X-Request-ID" json:"-"`
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"/things/"+pathID.String()+"?filter="+filterID.String()+"&related="+pathID.String()+"&related="+relatedID.String(),
		nil,
	)
	request.SetPathValue("id", pathID.String())
	request.Header.Set("X-Request-ID", requestID.String())

	got, err := bindInput[input](request, buildInputMetadata[input]("/things/{id}", newValidatorRegistry()))
	if err != nil {
		t.Fatalf("bindInput() error = %v", err)
	}
	want := input{
		ID:        pathID,
		Filter:    filterID,
		Related:   []uuid.UUID{pathID, relatedID},
		RequestID: requestID,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("input = %#v, want %#v", got, want)
	}
}

func TestBindInputRejectsInvalidUUIDParameter(t *testing.T) {
	type input struct {
		ID uuid.UUID `query:"id" json:"-"`
	}
	request := httptest.NewRequest(http.MethodGet, "/things?id=not-a-uuid", nil)

	_, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
	problem, ok := errors.AsType[*problem](err)
	if !ok {
		t.Fatalf("error = %T, want *problem", err)
	}
	if problem.Status != http.StatusBadRequest || problem.Detail != `invalid query parameter "id"` {
		t.Errorf("problem = %#v", problem)
	}
}

func TestBindInputLeavesAbsentQueryAndHeaderParametersAtZeroValue(t *testing.T) {
	type input struct {
		Page    int  `query:"page" json:"-"`
		Preview bool `header:"X-Preview" json:"-"`
	}
	request := httptest.NewRequest(http.MethodGet, "/things", nil)

	got, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
	if err != nil {
		t.Fatalf("bindInput() error = %v", err)
	}
	if got.Page != 0 || got.Preview {
		t.Errorf("input = %#v, want zero value", got)
	}
}

func TestBindInputCollectsRepeatedQueryValuesIntoSlices(t *testing.T) {
	type input struct {
		Tags      []string `query:"tag" json:"-"`
		Pages     []int    `query:"page" json:"-"`
		Public    []bool   `query:"public" json:"-"`
		Revisions []uint   `query:"revision" json:"-"`
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"/things?tag=go&tag=http&page=-1&page=2&public=true&public=false&revision=1&revision=3",
		nil,
	)

	got, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
	if err != nil {
		t.Fatalf("bindInput() error = %v", err)
	}
	want := input{
		Tags:      []string{"go", "http"},
		Pages:     []int{-1, 2},
		Public:    []bool{true, false},
		Revisions: []uint{1, 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("input = %#v, want %#v", got, want)
	}
}

func TestBindInputDoesNotSplitCommaSeparatedQueryValues(t *testing.T) {
	type input struct {
		Tags []string `query:"tag" json:"-"`
	}
	request := httptest.NewRequest(http.MethodGet, "/things?tag=go,http", nil)

	got, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
	if err != nil {
		t.Fatalf("bindInput() error = %v", err)
	}
	want := []string{"go,http"}
	if !reflect.DeepEqual(got.Tags, want) {
		t.Errorf("tags = %#v, want %#v", got.Tags, want)
	}
}

func TestBindInputRejectsRepeatedScalarQueryParameter(t *testing.T) {
	type input struct {
		Tag string `query:"tag" json:"-"`
	}
	request := httptest.NewRequest(http.MethodGet, "/things?tag=go&tag=http", nil)

	_, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
	problem, ok := errors.AsType[*problem](err)
	if !ok {
		t.Fatalf("error = %T, want *problem", err)
	}
	if problem.Status != http.StatusBadRequest || problem.Detail != `query parameter "tag" must not be repeated` {
		t.Errorf("problem = %#v", problem)
	}
}

func TestBindInputRejectsInvalidQuerySliceElement(t *testing.T) {
	type input struct {
		Pages []int `query:"page" json:"-"`
	}
	request := httptest.NewRequest(http.MethodGet, "/things?page=1&page=many", nil)

	_, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
	problem, ok := errors.AsType[*problem](err)
	if !ok {
		t.Fatalf("error = %T, want *problem", err)
	}
	if problem.Status != http.StatusBadRequest || problem.Detail != `invalid query parameter "page"` {
		t.Errorf("problem = %#v", problem)
	}
}

func TestBindInputReportsInvalidQueryAndHeaderParameters(t *testing.T) {
	tests := []struct {
		name       string
		request    *http.Request
		bind       func(*http.Request) error
		wantDetail string
	}{
		{
			name:    "query",
			request: httptest.NewRequest(http.MethodGet, "/things?page=many", nil),
			bind: func(request *http.Request) error {
				type input struct {
					Page int `query:"page" json:"-"`
				}
				_, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
				return err
			},
			wantDetail: `invalid query parameter "page"`,
		},
		{
			name: "header",
			request: func() *http.Request {
				request := httptest.NewRequest(http.MethodGet, "/things", nil)
				request.Header.Set("X-Preview", "sometimes")
				return request
			}(),
			bind: func(request *http.Request) error {
				type input struct {
					Preview bool `header:"X-Preview" json:"-"`
				}
				_, err := bindInput[input](request, buildInputMetadata[input]("/things", newValidatorRegistry()))
				return err
			},
			wantDetail: `invalid header parameter "X-Preview"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			problem, ok := errors.AsType[*problem](test.bind(test.request))
			if !ok {
				t.Fatal("error is not an HTTP problem")
			}
			if problem.Status != http.StatusBadRequest || problem.Detail != test.wantDetail {
				t.Errorf("problem = %#v", problem)
			}
		})
	}
}

func TestSetParameterValueSupportsScalarTypes(t *testing.T) {
	tests := []struct {
		name  string
		type_ reflect.Type
		value string
		want  any
	}{
		{name: "string", type_: reflect.TypeFor[string](), value: "abc", want: "abc"},
		{name: "boolean", type_: reflect.TypeFor[bool](), value: "true", want: true},
		{name: "signed integer", type_: reflect.TypeFor[int32](), value: "-42", want: int32(-42)},
		{name: "unsigned integer", type_: reflect.TypeFor[uint16](), value: "42", want: uint16(42)},
		{
			name:  "UUID",
			type_: reflect.TypeFor[uuid.UUID](),
			value: "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
			want:  uuid.MustParse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := reflect.New(test.type_).Elem()
			if err := setParameterValue(field, test.value); err != nil {
				t.Fatalf("setParameterValue() error = %v", err)
			}
			if got := field.Interface(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("value = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestSetParameterValueRejectsInvalidScalars(t *testing.T) {
	tests := []struct {
		name  string
		type_ reflect.Type
	}{
		{name: "boolean", type_: reflect.TypeFor[bool]()},
		{name: "signed integer", type_: reflect.TypeFor[int]()},
		{name: "unsigned integer", type_: reflect.TypeFor[uint]()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := reflect.New(test.type_).Elem()
			if err := setParameterValue(field, "not-a-value"); err == nil {
				t.Error("setParameterValue() error = nil")
			}
		})
	}
}

func TestBindInputReportsInvalidPathParameter(t *testing.T) {
	type input struct {
		ID int `path:"id" json:"-"`
	}
	request := httptest.NewRequest(http.MethodGet, "/things/not-a-number", nil)
	request.SetPathValue("id", "not-a-number")

	_, err := bindInput[input](request, buildInputMetadata[input]("/things/{id}", newValidatorRegistry()))
	problem, ok := errors.AsType[*problem](err)
	if !ok {
		t.Fatalf("error = %T, want *problem", err)
	}
	if problem.Status != http.StatusBadRequest || problem.Detail != `invalid path parameter "id"` {
		t.Errorf("problem = %#v", problem)
	}
}

func TestSetParameterValuePanicsForUnsupportedType(t *testing.T) {
	assertPanics(t, func() {
		_ = setParameterValue(reflect.ValueOf(new(float64)).Elem(), "1.5")
	})
}
