package generator

import (
	"testing"
)

// FuzzToSnakeCase fuzzes the toSnakeCase function with arbitrary strings
// to ensure it never panics on any input.
func FuzzToSnakeCase(f *testing.F) {
	// Seed corpus
	f.Add("SimpleCase")
	f.Add("camelCase")
	f.Add("PascalCase")
	f.Add("ALLCAPS")
	f.Add("with_underscores")
	f.Add("mixedCASE_Stuff")
	f.Add("HTTPSConnection")
	f.Add("getHTTPResponse")
	f.Add("")
	f.Add("a")
	f.Add("ABC")
	f.Add("already_snake_case")
	f.Add("with123Numbers")
	f.Add("123StartsWithNumber")
	f.Add("unicode_日本語")
	f.Add("   spaces   ")
	f.Add("special!@#$%chars")

	f.Fuzz(func(t *testing.T, input string) {
		// Should never panic
		result := toSnakeCase(input)
		// Result should not be longer than a reasonable multiple of input
		if len(result) > len(input)*3+10 {
			t.Errorf("toSnakeCase(%q) produced unexpectedly long output: %q", input, result)
		}
	})
}

// FuzzEnumPrefix fuzzes the enumPrefix function.
func FuzzEnumPrefix(f *testing.F) {
	f.Add("TaskState")
	f.Add("Status")
	f.Add("MyEnum")
	f.Add("")
	f.Add("A")
	f.Add("ALLCAPS")
	f.Add("with_underscores")
	f.Add("lower")

	f.Fuzz(func(t *testing.T, input string) {
		// Should never panic
		_ = enumPrefix(input)
	})
}

// FuzzCleanComment fuzzes the cleanComment function.
func FuzzCleanComment(f *testing.F) {
	f.Add("// This is a comment")
	f.Add("/* block comment */")
	f.Add(" leading whitespace")
	f.Add("multi\nline\ncomment")
	f.Add("")
	f.Add("no comment markers")
	f.Add("  //  double space")
	f.Add("tab\there")

	f.Fuzz(func(t *testing.T, input string) {
		result := cleanComment(input)
		// Result should never be longer than input
		if len(result) > len(input) {
			t.Errorf("cleanComment(%q) produced longer output: %q", input, result)
		}
	})
}

// FuzzEscapeString fuzzes the escapeString function.
func FuzzEscapeString(f *testing.F) {
	f.Add("hello world")
	f.Add(`has "quotes"`)
	f.Add("has 'quotes'")
	f.Add("has\nnewline")
	f.Add("has\ttab")
	f.Add(`has \backslash`)
	f.Add("")
	f.Add("unicode: 日本語")
	f.Add("null: \x00 byte")

	f.Fuzz(func(t *testing.T, input string) {
		// Should never panic
		_ = escapeString(input)
	})
}

// FuzzOutputFilename fuzzes the outputFilename function with various paths and options.
func FuzzOutputFilename(f *testing.F) {
	f.Add("a2a.proto", "", false)
	f.Add("path/to/file.proto", "", false)
	f.Add("test.proto", "custom.py", false)
	f.Add("test.proto", "", true)
	f.Add("", "", false)
	f.Add(".proto", "", false)
	f.Add("no_extension", "", false)
	f.Add("deep/nested/path/file.proto", "output.py", true)

	f.Fuzz(func(t *testing.T, protoPath string, outputFile string, stripSuffix bool) {
		opts := &Options{
			OutputFile:       outputFile,
			StripProtoSuffix: stripSuffix,
		}
		// Should never panic
		_ = outputFilename(protoPath, opts)
	})
}
