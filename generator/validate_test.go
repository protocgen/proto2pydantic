package generator

import (
	"testing"
)

func TestFieldConstraints_HasConstraints(t *testing.T) {
	tests := []struct {
		name     string
		c        FieldConstraints
		expected bool
	}{
		{"empty", FieldConstraints{}, false},
		{"min_length", FieldConstraints{MinLength: ptrUint64(1)}, true},
		{"max_length", FieldConstraints{MaxLength: ptrUint64(100)}, true},
		{"pattern", FieldConstraints{Pattern: "^[a-z]+$"}, true},
		{"email", FieldConstraints{Email: true}, true},
		{"uuid", FieldConstraints{UUID: true}, true},
		{"uri", FieldConstraints{URI: true}, true},
		{"gt", FieldConstraints{Gt: "0"}, true},
		{"gte", FieldConstraints{Gte: "1"}, true},
		{"lt", FieldConstraints{Lt: "100"}, true},
		{"lte", FieldConstraints{Lte: "99"}, true},
		{"min_items", FieldConstraints{MinItems: ptrUint64(1)}, true},
		{"max_items", FieldConstraints{MaxItems: ptrUint64(10)}, true},
		{"const", FieldConstraints{Const: "hello"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.HasConstraints()
			if got != tt.expected {
				t.Errorf("HasConstraints() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFieldConstraints_ToPydanticArgs(t *testing.T) {
	tests := []struct {
		name     string
		c        FieldConstraints
		expected string
	}{
		{
			"min_length only",
			FieldConstraints{MinLength: ptrUint64(1)},
			"min_length=1",
		},
		{
			"max_length only",
			FieldConstraints{MaxLength: ptrUint64(255)},
			"max_length=255",
		},
		{
			"min and max length",
			FieldConstraints{MinLength: ptrUint64(1), MaxLength: ptrUint64(100)},
			"min_length=1, max_length=100",
		},
		{
			"pattern",
			FieldConstraints{Pattern: "^[a-z]+$"},
			"pattern='^[a-z]+$'",
		},
		{
			"numeric gte and lte",
			FieldConstraints{Gte: "0", Lte: "150"},
			"ge=0, le=150",
		},
		{
			"numeric gt and lt",
			FieldConstraints{Gt: "0", Lt: "100"},
			"gt=0, lt=100",
		},
		{
			"repeated min and max items",
			FieldConstraints{MinItems: ptrUint64(1), MaxItems: ptrUint64(10)},
			"min_length=1, max_length=10",
		},
		{
			"empty constraints",
			FieldConstraints{},
			"",
		},
		{
			"combined string constraints",
			FieldConstraints{MinLength: ptrUint64(5), MaxLength: ptrUint64(50), Pattern: "^[A-Z]"},
			"min_length=5, max_length=50, pattern='^[A-Z]'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.ToPydanticArgs()
			if got != tt.expected {
				t.Errorf("ToPydanticArgs() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func ptrUint64(v uint64) *uint64 {
	return &v
}
