package amigo

import "testing"

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
					ID float64 `path:"id" json:"-"`
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
					Filters []float64 `query:"filter" json:"-"`
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
