package amigo

import (
	"reflect"
	"testing"
	"time"
	"uuid"
)

type EmbeddedInputFields struct {
	Name string `json:"name"`
}

func TestBuildInputMetadataRegistersCompleteBody(t *testing.T) {
	type input struct {
		ID         string    `path:"id" json:"-"`
		Search     string    `query:"search" json:"-"`
		RequestID  string    `header:"X-Request-ID" json:"-"`
		Name       string    `json:"name" validate:"required"`
		CreatedAt  time.Time `json:"created_at"`
		ExternalID uuid.UUID `json:"external_id,omitempty"`
		Enabled    bool
		Ignored    string `json:"-"`
	}

	metadata := buildInputMetadata[input]("/things/{id}", newValidatorRegistry())

	if metadata.body.isEmpty() {
		t.Fatal("body metadata is empty")
	}
	want := []bodyField{
		{name: "name", fieldID: 3, fieldIndex: []int{3}, fieldType: reflect.TypeFor[string](), jsonTag: "name"},
		{name: "created_at", fieldID: 4, fieldIndex: []int{4}, fieldType: reflect.TypeFor[time.Time](), jsonTag: "created_at"},
		{name: "external_id", fieldID: 5, fieldIndex: []int{5}, fieldType: reflect.TypeFor[uuid.UUID](), jsonTag: "external_id,omitempty"},
		{name: "Enabled", fieldID: 6, fieldIndex: []int{6}, fieldType: reflect.TypeFor[bool](), jsonTag: ""},
	}
	if !reflect.DeepEqual(metadata.body.fields, want) {
		t.Errorf("body fields = %#v, want %#v", metadata.body.fields, want)
	}
	if len(metadata.body.indexByName) != len(want) {
		t.Fatalf("body name index has %d entries, want %d", len(metadata.body.indexByName), len(want))
	}
	for index, field := range want {
		got, exists := metadata.body.indexByName[field.name]
		if !exists || got != index {
			t.Errorf("body index for %q = %d, want %d", field.name, got, index)
		}
	}
	if len(metadata.validations) != 1 {
		t.Errorf("validations = %d, want 1", len(metadata.validations))
	}
}

func TestBuildInputMetadataRecognizesMissingBody(t *testing.T) {
	type input struct {
		ID string `path:"id" json:"-"`
	}

	metadata := buildInputMetadata[input]("/things/{id}", newValidatorRegistry())

	if !metadata.body.isEmpty() {
		t.Errorf("body fields = %#v, want empty", metadata.body.fields)
	}
}

func TestBodyMetadataRejectsDuplicateJSONName(t *testing.T) {
	field := reflect.TypeFor[struct {
		Value string `json:"value"`
	}]().Field(0)
	body := newBodyMetadata()
	body.add(field, 0, "value")

	assertPanics(t, func() {
		body.add(field, 1, "value")
	})
}

func TestBodyFieldNameRejectsJSONTagOnUnexportedField(t *testing.T) {
	field := reflect.StructField{
		Name:    "value",
		PkgPath: "github.com/vekio/amigo",
		Type:    reflect.TypeFor[string](),
		Tag:     `json:"value"`,
	}

	assertPanics(t, func() {
		_, _ = bodyFieldName(field)
	})
}

func TestBuildInputMetadataRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		build func()
	}{
		{
			name:  "non-struct input",
			build: func() { buildInputMetadata[string]("/things", newValidatorRegistry()) },
		},
		{
			name: "unexported field",
			build: func() {
				buildInputMetadata[struct {
					id string `path:"id" json:"-"`
				}]("/things/{id}", newValidatorRegistry())
			},
		},
		{
			name: "empty path tag",
			build: func() {
				buildInputMetadata[struct {
					ID string `path:"" json:"-"`
				}]("/things/{id}", newValidatorRegistry())
			},
		},
		{
			name: "unknown path name",
			build: func() {
				buildInputMetadata[struct {
					ID string `path:"other" json:"-"`
				}]("/things/{id}", newValidatorRegistry())
			},
		},
		{
			name: "duplicate path binding",
			build: func() {
				buildInputMetadata[struct {
					First  string `path:"id" json:"-"`
					Second string `path:"id" json:"-"`
				}]("/things/{id}", newValidatorRegistry())
			},
		},
		{
			name: "unsupported path type",
			build: func() {
				buildInputMetadata[struct {
					ID complex64 `path:"id" json:"-"`
				}]("/things/{id}", newValidatorRegistry())
			},
		},
		{
			name: "path field included in JSON",
			build: func() {
				buildInputMetadata[struct {
					ID string `path:"id" json:"id"`
				}]("/things/{id}", newValidatorRegistry())
			},
		},
		{
			name:  "missing path binding",
			build: func() { buildInputMetadata[struct{}]("/things/{id}", newValidatorRegistry()) },
		},
		{
			name: "multiple parameter sources",
			build: func() {
				buildInputMetadata[struct {
					Value string `query:"value" header:"X-Value" json:"-"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "unexported query field",
			build: func() {
				buildInputMetadata[struct {
					limit int `query:"limit" json:"-"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "empty query tag",
			build: func() {
				buildInputMetadata[struct {
					Limit int `query:"" json:"-"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "duplicate query binding",
			build: func() {
				buildInputMetadata[struct {
					First  int `query:"limit" json:"-"`
					Second int `query:"limit" json:"-"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "unsupported query type",
			build: func() {
				buildInputMetadata[struct {
					Filters []complex64 `query:"filter" json:"-"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "slice path parameter",
			build: func() {
				buildInputMetadata[struct {
					IDs []int `path:"id" json:"-"`
				}]("/things/{id}", newValidatorRegistry())
			},
		},
		{
			name: "slice header parameter",
			build: func() {
				buildInputMetadata[struct {
					Values []string `header:"X-Value" json:"-"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "query field included in JSON",
			build: func() {
				buildInputMetadata[struct {
					Limit int `query:"limit" json:"limit"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "duplicate case-insensitive header binding",
			build: func() {
				buildInputMetadata[struct {
					First  string `header:"X-Request-ID" json:"-"`
					Second string `header:"x-request-id" json:"-"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "parameter with non-exact ignored JSON tag",
			build: func() {
				buildInputMetadata[struct {
					ID string `path:"id" json:"-,omitempty"`
				}]("/things/{id}", newValidatorRegistry())
			},
		},
		{
			name: "implicitly embedded JSON field",
			build: func() {
				buildInputMetadata[struct {
					EmbeddedInputFields
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "explicitly embedded JSON field",
			build: func() {
				buildInputMetadata[struct {
					Details EmbeddedInputFields `json:",embed"`
				}]("/things", newValidatorRegistry())
			},
		},
		{
			name: "case-insensitive JSON field",
			build: func() {
				buildInputMetadata[struct {
					Name string `json:"name,case:ignore"`
				}]("/things", newValidatorRegistry())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertPanics(t, test.build)
		})
	}
}

func TestBuildOutputMetadataRejectsInvalidOutput(t *testing.T) {
	tests := []struct {
		name  string
		build func()
	}{
		{
			name:  "non-struct output",
			build: func() { buildOutputMetadata[string]() },
		},
		{
			name: "unexported header field",
			build: func() {
				buildOutputMetadata[struct {
					location string `header:"Location" json:"-"`
				}]()
			},
		},
		{
			name: "empty header tag",
			build: func() {
				buildOutputMetadata[struct {
					Location string `header:"" json:"-"`
				}]()
			},
		},
		{
			name: "non-string header",
			build: func() {
				buildOutputMetadata[struct {
					RetryAfter int `header:"Retry-After" json:"-"`
				}]()
			},
		},
		{
			name: "header included in JSON",
			build: func() {
				buildOutputMetadata[struct {
					Location string `header:"Location" json:"location"`
				}]()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertPanics(t, test.build)
		})
	}
}

func TestRoutePathNamesRecognizesWildcards(t *testing.T) {
	names := routePathNames("/files/{bucket}/{path...}/{$}")
	if _, ok := names["bucket"]; !ok {
		t.Error("bucket wildcard was not found")
	}
	if _, ok := names["path"]; !ok {
		t.Error("catch-all wildcard was not found")
	}
	if _, ok := names["$"]; ok {
		t.Error("end-of-path marker must not be bound")
	}
}
