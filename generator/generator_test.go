package generator

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"message_id", "message_id"},
		{"messageId", "message_id"},
		{"MessageId", "message_id"},
		{"url", "url"},
		{"URL", "url"},
		{"httpAuth", "http_auth"},
		{"id", "id"},
		{"a", "a"},
		{"", ""},
		{"pkceRequired", "pkce_required"},
		{"openIdConnectUrl", "open_id_connect_url"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEnumPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TaskState", "TASK_STATE_"},
		{"Role", "ROLE_"},
		{"SexType", "SEX_TYPE_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := enumPrefix(tt.input)
			if got != tt.expected {
				t.Errorf("enumPrefix(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestScalarPythonType(t *testing.T) {
	tests := []struct {
		name     string
		kind     protoreflect.Kind
		expected string
	}{
		{"bool", protoreflect.BoolKind, "bool"},
		{"int32", protoreflect.Int32Kind, "int"},
		{"int64", protoreflect.Int64Kind, "int"},
		{"uint32", protoreflect.Uint32Kind, "int"},
		{"uint64", protoreflect.Uint64Kind, "int"},
		{"string", protoreflect.StringKind, "str"},
		{"float", protoreflect.FloatKind, "float"},
		{"double", protoreflect.DoubleKind, "float"},
		{"bytes", protoreflect.BytesKind, "bytes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scalarPythonType(tt.kind)
			if got != tt.expected {
				t.Errorf("scalarPythonType(%v) = %q, want %q", tt.kind, got, tt.expected)
			}
		})
	}
}

func TestScalarDefault(t *testing.T) {
	tests := []struct {
		name     string
		kind     protoreflect.Kind
		expected string
	}{
		{"bool", protoreflect.BoolKind, "False"},
		{"int32", protoreflect.Int32Kind, "0"},
		{"int64", protoreflect.Int64Kind, "0"},
		{"string", protoreflect.StringKind, "''"},
		{"float", protoreflect.FloatKind, "0.0"},
		{"double", protoreflect.DoubleKind, "0.0"},
		{"bytes", protoreflect.BytesKind, "b''"},
		{"message", protoreflect.MessageKind, "None"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scalarDefault(tt.kind)
			if got != tt.expected {
				t.Errorf("scalarDefault(%v) = %q, want %q", tt.kind, got, tt.expected)
			}
		})
	}
}

func TestCleanComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"simple", " A simple comment.", "A simple comment."},
		{"with slashes", "// A comment with slashes.", "A comment with slashes."},
		{"multiline", " First line.\n Second line.", "First line. Second line."},
		{"proto comment", "// First line.\n// Second line.", "First line. Second line."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanComment(tt.input)
			if got != tt.expected {
				t.Errorf("cleanComment(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"no quotes", "no quotes"},
		{"it's a test", "it\\'s a test"},
		{"'quoted'", "\\'quoted\\'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeString(tt.input)
			if got != tt.expected {
				t.Errorf("escapeString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestOutputFilename(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		opts     *Options
		expected string
	}{
		{
			"default",
			"a2a.proto",
			&Options{},
			"a2a_pb2_pydantic.py",
		},
		{
			"custom output file",
			"a2a.proto",
			&Options{OutputFile: "types.py"},
			"types.py",
		},
		{
			"strip proto suffix",
			"a2a.proto",
			&Options{StripProtoSuffix: true},
			"a2a.py",
		},
		{
			"output file takes precedence over strip",
			"a2a.proto",
			&Options{OutputFile: "models.py", StripProtoSuffix: true},
			"models.py",
		},
		{
			"nested path",
			"foo/bar/service.proto",
			&Options{},
			"foo/bar/service_pb2_pydantic.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outputFilename(tt.path, tt.opts)
			if got != tt.expected {
				t.Errorf("outputFilename(%q, %+v) = %q, want %q", tt.path, tt.opts, got, tt.expected)
			}
		})
	}
}

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name     string
		models   []PydanticModel
		expected []string // expected order of model names
	}{
		{
			"linear chain: C depends on B depends on A",
			[]PydanticModel{
				{Name: "C", Fields: []PydanticField{{PythonType: "B"}}},
				{Name: "B", Fields: []PydanticField{{PythonType: "A"}}},
				{Name: "A", Fields: []PydanticField{{PythonType: "str"}}},
			},
			[]string{"A", "B", "C"},
		},
		{
			"already sorted",
			[]PydanticModel{
				{Name: "A", Fields: []PydanticField{{PythonType: "str"}}},
				{Name: "B", Fields: []PydanticField{{PythonType: "A"}}},
			},
			[]string{"A", "B"},
		},
		{
			"independent models preserve order",
			[]PydanticModel{
				{Name: "X", Fields: []PydanticField{{PythonType: "str"}}},
				{Name: "Y", Fields: []PydanticField{{PythonType: "int"}}},
				{Name: "Z", Fields: []PydanticField{{PythonType: "bool"}}},
			},
			[]string{"X", "Y", "Z"},
		},
		{
			"list reference",
			[]PydanticModel{
				{Name: "Parent", Fields: []PydanticField{{PythonType: "list[Child]"}}},
				{Name: "Child", Fields: []PydanticField{{PythonType: "str"}}},
			},
			[]string{"Child", "Parent"},
		},
		{
			"optional reference",
			[]PydanticModel{
				{Name: "Outer", Fields: []PydanticField{{PythonType: "Inner | None"}}},
				{Name: "Inner", Fields: []PydanticField{{PythonType: "str"}}},
			},
			[]string{"Inner", "Outer"},
		},
		{
			"single model",
			[]PydanticModel{
				{Name: "Solo", Fields: []PydanticField{{PythonType: "str"}}},
			},
			[]string{"Solo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := topologicalSort(tt.models)
			if len(sorted) != len(tt.expected) {
				t.Fatalf("got %d models, want %d", len(sorted), len(tt.expected))
			}
			for i, name := range tt.expected {
				if sorted[i].Name != name {
					var gotNames []string
					for _, m := range sorted {
						gotNames = append(gotNames, m.Name)
					}
					t.Errorf("got order %v, want %v", gotNames, tt.expected)
					break
				}
			}
		})
	}
}
