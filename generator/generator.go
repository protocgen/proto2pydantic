// Package generator implements the core logic for converting protobuf
// descriptors into Pydantic v2 BaseModel Python source code.
package generator

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

// Options configures the code generation behavior.
type Options struct {
	BaseClass        string // custom base class (default: "BaseModel")
	AliasGenerator   string // alias generator for model_config ("camel")
	OutputFile       string // override output filename
	StripProtoSuffix bool   // strip _pb2_pydantic from filename
	Description      string // override module-level docstring
	EnumStyle        string // enum style: "" (default pythonic) or "raw" (proto names)
	Preset           string // preset: "a2a" sets alias_generator=camel + enum_style=raw
}

// applyPreset sets default options based on the chosen preset.
func (o *Options) applyPreset() {
	if o.Preset == "a2a" {
		if o.AliasGenerator == "" {
			o.AliasGenerator = "camel"
		}
		if o.EnumStyle == "" {
			o.EnumStyle = "raw"
		}
	}
}

// PydanticFile represents the full generated Python file.
type PydanticFile struct {
	SourceProto string
	Enums       []PydanticEnum
	Models      []PydanticModel
	Imports     map[string]bool // tracks needed imports like "datetime"
	Opts        *Options
}

// PydanticEnum represents a Python Enum class.
type PydanticEnum struct {
	Name        string
	Description string
	Values      []PydanticEnumValue
}

// PydanticEnumValue represents a single enum member.
type PydanticEnumValue struct {
	Name  string // Python-friendly name (lowercase, no prefix)
	Value string // string value for str enum
}

// PydanticModel represents a Pydantic BaseModel class.
type PydanticModel struct {
	Name            string
	Description     string
	Fields          []PydanticField
	OneOfs          []PydanticOneOf
	TimestampFields []string // field names that are datetime (need RFC 3339 serializer)
	BytesFields     []string // field names that are bytes (need base64 serializer)
}

// PydanticOneOf represents a oneof group rendered as a union type.
type PydanticOneOf struct {
	Name       string
	FieldName  string // the Python field name
	PythonType string // e.g. "TextPart | FilePart | DataPart | None"
}

// PydanticField represents a single field in a Pydantic model.
type PydanticField struct {
	Name        string            // snake_case Python name
	PythonType  string            // e.g. "str", "int", "list[str]", "MyModel"
	Required    bool              // from field_behavior REQUIRED
	Optional    bool              // from proto3 optional keyword
	OutputOnly  bool              // from field_behavior OUTPUT_ONLY
	IsList      bool              // repeated field
	IsMap       bool              // map field
	Default     string            // default value string, empty if required
	Description string            // from proto comments
	OneOfName   string            // if part of a oneof, the oneof group name
	Constraints *FieldConstraints // from buf/validate rules
}

// GenerateFile generates a Python file with Pydantic models for the given proto file.
func GenerateFile(gen *protogen.Plugin, file *protogen.File, opts *Options) error {
	// Apply preset defaults before processing
	opts.applyPreset()

	pyFile := &PydanticFile{
		SourceProto: file.Desc.Path(),
		Imports:     make(map[string]bool),
		Opts:        opts,
	}

	// Process enums
	for _, enum := range file.Enums {
		pyEnum := processEnum(enum, opts)
		pyFile.Enums = append(pyFile.Enums, pyEnum)
	}

	// Process messages (models)
	for _, msg := range file.Messages {
		pyModel := processMessage(msg, pyFile)
		pyFile.Models = append(pyFile.Models, pyModel)

		// Also process nested enums and messages
		for _, nestedEnum := range msg.Enums {
			pyFile.Enums = append(pyFile.Enums, processEnum(nestedEnum, opts))
		}
	}

	// Topological sort: ensure models are defined before they are referenced
	pyFile.Models = topologicalSort(pyFile.Models)

	// Determine output filename
	filename := outputFilename(file.Desc.Path(), opts)
	g := gen.NewGeneratedFile(filename, "")
	writeFile(g, pyFile)
	return nil
}

// outputFilename determines the Python output filename based on options.
func outputFilename(protoPath string, opts *Options) string {
	if opts.OutputFile != "" {
		return opts.OutputFile
	}
	base := strings.TrimSuffix(protoPath, ".proto")
	if opts.StripProtoSuffix {
		// a2a.proto -> a2a.py
		return base + ".py"
	}
	// a2a.proto -> a2a_pb2_pydantic.py
	return base + "_pb2_pydantic.py"
}

// processEnum converts a protobuf enum descriptor to a PydanticEnum.
func processEnum(enum *protogen.Enum, opts *Options) PydanticEnum {
	pyEnum := PydanticEnum{
		Name:        string(enum.Desc.Name()),
		Description: cleanComment(enum.Comments.Leading.String()),
	}

	for _, val := range enum.Values {
		name := string(val.Desc.Name())

		if opts.EnumStyle == "raw" {
			// Raw style: preserve original proto names including UNSPECIFIED
			pyEnum.Values = append(pyEnum.Values, PydanticEnumValue{
				Name:  name,
				Value: name,
			})
		} else {
			// Default style: strip prefix, lowercase, skip UNSPECIFIED
			if strings.HasSuffix(name, "_UNSPECIFIED") {
				continue
			}
			prefix := enumPrefix(string(enum.Desc.Name()))
			pythonName := strings.TrimPrefix(name, prefix)
			pythonName = strings.ToLower(pythonName)

			pyEnum.Values = append(pyEnum.Values, PydanticEnumValue{
				Name:  pythonName,
				Value: pythonName,
			})
		}
	}

	return pyEnum
}

// processMessage converts a protobuf message descriptor to a PydanticModel.
func processMessage(msg *protogen.Message, pyFile *PydanticFile) PydanticModel {
	pyModel := PydanticModel{
		Name:        string(msg.Desc.Name()),
		Description: cleanComment(msg.Comments.Leading.String()),
	}

	// Track which fields belong to oneofs
	oneofFields := make(map[string][]PydanticField) // oneofName -> fields

	for _, field := range msg.Fields {
		pyField := processField(field, pyFile)

		// Check if field is part of a oneof
		if field.Desc.ContainingOneof() != nil && !field.Desc.HasOptionalKeyword() {
			oneofName := string(field.Desc.ContainingOneof().Name())
			pyField.OneOfName = oneofName
			oneofFields[oneofName] = append(oneofFields[oneofName], pyField)
			continue // Don't add as a regular field
		}

		pyModel.Fields = append(pyModel.Fields, pyField)

		// Track fields that need ProtoJSON serializers
		if field.Desc.Kind() == protoreflect.MessageKind && field.Desc.Message() != nil {
			fullName := string(field.Desc.Message().FullName())
			if fullName == "google.protobuf.Timestamp" {
				pyModel.TimestampFields = append(pyModel.TimestampFields, pyField.Name)
			}
		}
		if field.Desc.Kind() == protoreflect.BytesKind {
			pyModel.BytesFields = append(pyModel.BytesFields, pyField.Name)
		}
	}

	// Process oneof groups into union types
	for _, oneof := range msg.Oneofs {
		// Skip synthetic oneofs (created by proto3 optional)
		if oneof.Desc.IsSynthetic() {
			continue
		}

		name := string(oneof.Desc.Name())
		fields := oneofFields[name]
		if len(fields) == 0 {
			continue
		}

		var types []string
		for _, f := range fields {
			types = append(types, f.PythonType)
		}

		pyModel.OneOfs = append(pyModel.OneOfs, PydanticOneOf{
			Name:       name,
			FieldName:  toSnakeCase(name),
			PythonType: strings.Join(types, " | ") + " | None",
		})
	}

	return pyModel
}

// processField converts a protobuf field descriptor to a PydanticField.
func processField(field *protogen.Field, pyFile *PydanticFile) PydanticField {
	pyField := PydanticField{
		Name:        toSnakeCase(string(field.Desc.Name())),
		Description: cleanComment(field.Comments.Leading.String()),
		Required:    isRequired(field),
		OutputOnly:  isOutputOnly(field),
		Optional:    field.Desc.HasOptionalKeyword(),
		IsList:      field.Desc.IsList(),
		IsMap:       field.Desc.IsMap(),
		Constraints: extractConstraints(field),
	}

	// buf.validate required also makes the field required
	if pyField.Constraints != nil && pyField.Constraints.ValidateRequired {
		pyField.Required = true
	}

	// Determine the Python type
	pyField.PythonType = pythonType(field, pyFile)

	// Wrap in list/optional as needed
	if pyField.IsMap {
		keyType := scalarPythonType(field.Desc.MapKey().Kind())
		valType := mapValuePythonType(field, pyFile)
		pyField.PythonType = fmt.Sprintf("dict[%s, %s]", keyType, valType)
	} else if pyField.IsList {
		pyField.PythonType = fmt.Sprintf("list[%s]", pyField.PythonType)
	}

	// Determine default value
	if pyField.Required {
		pyField.Default = "" // no default = required in Pydantic
	} else if pyField.Optional {
		pyField.PythonType = pyField.PythonType + " | None"
		pyField.Default = "None"
	} else if pyField.IsList {
		pyField.Default = "None"
		pyField.PythonType = pyField.PythonType + " | None"
	} else if pyField.IsMap {
		pyField.Default = "None"
		pyField.PythonType = pyField.PythonType + " | None"
	} else {
		pyField.Default = scalarDefault(field.Desc.Kind())
	}

	return pyField
}

// getFieldBehaviors extracts google.api.field_behavior annotations from a field.
func getFieldBehaviors(field *protogen.Field) []annotations.FieldBehavior {
	opts := field.Desc.Options()
	if opts == nil {
		return nil
	}

	fieldOpts, ok := opts.(*descriptorpb.FieldOptions)
	if !ok {
		return nil
	}

	if !proto.HasExtension(fieldOpts, annotations.E_FieldBehavior) {
		return nil
	}

	return proto.GetExtension(fieldOpts, annotations.E_FieldBehavior).([]annotations.FieldBehavior)
}

// isRequired checks if a field has the google.api.field_behavior REQUIRED annotation.
func isRequired(field *protogen.Field) bool {
	for _, b := range getFieldBehaviors(field) {
		if b == annotations.FieldBehavior_REQUIRED {
			return true
		}
	}
	return false
}

// isOutputOnly checks if a field has the google.api.field_behavior OUTPUT_ONLY annotation.
func isOutputOnly(field *protogen.Field) bool {
	for _, b := range getFieldBehaviors(field) {
		if b == annotations.FieldBehavior_OUTPUT_ONLY {
			return true
		}
	}
	return false
}

// pythonType returns the base Python type string for a protobuf field.
func pythonType(field *protogen.Field, pyFile *PydanticFile) string {
	if field.Desc.IsMap() {
		// Maps are handled by the caller
		return ""
	}

	switch field.Desc.Kind() {
	case protoreflect.MessageKind:
		return messagePythonType(field, pyFile)
	case protoreflect.EnumKind:
		return string(field.Desc.Enum().Name())
	default:
		return scalarPythonType(field.Desc.Kind())
	}
}

// messagePythonType returns the Python type for a message field,
// handling well-known types specially.
func messagePythonType(field *protogen.Field, pyFile *PydanticFile) string {
	fullName := string(field.Desc.Message().FullName())

	switch fullName {
	case "google.protobuf.Struct":
		pyFile.Imports["Any"] = true
		return "dict[str, Any]"
	case "google.protobuf.Value":
		pyFile.Imports["Any"] = true
		return "Any"
	case "google.protobuf.Timestamp":
		pyFile.Imports["datetime"] = true
		return "datetime"
	case "google.protobuf.Duration":
		pyFile.Imports["timedelta"] = true
		return "timedelta"
	case "google.protobuf.Empty":
		return "None"
	case "google.protobuf.StringValue":
		return "str | None"
	case "google.protobuf.Int32Value", "google.protobuf.Int64Value",
		"google.protobuf.UInt32Value", "google.protobuf.UInt64Value":
		return "int | None"
	case "google.protobuf.FloatValue", "google.protobuf.DoubleValue":
		return "float | None"
	case "google.protobuf.BoolValue":
		return "bool | None"
	case "google.protobuf.BytesValue":
		return "bytes | None"
	default:
		return string(field.Desc.Message().Name())
	}
}

// mapValuePythonType returns the Python type for the value of a map field.
func mapValuePythonType(field *protogen.Field, pyFile *PydanticFile) string {
	valDesc := field.Desc.MapValue()
	switch valDesc.Kind() {
	case protoreflect.MessageKind:
		fullName := string(valDesc.Message().FullName())
		switch fullName {
		case "google.protobuf.Struct":
			pyFile.Imports["Any"] = true
			return "dict[str, Any]"
		case "google.protobuf.Value":
			pyFile.Imports["Any"] = true
			return "Any"
		default:
			return string(valDesc.Message().Name())
		}
	case protoreflect.EnumKind:
		return string(valDesc.Enum().Name())
	default:
		return scalarPythonType(valDesc.Kind())
	}
}

// scalarPythonType maps proto scalar types to Python types.
func scalarPythonType(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "int"
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return "float"
	case protoreflect.StringKind:
		return "str"
	case protoreflect.BytesKind:
		return "bytes"
	default:
		return "Any"
	}
}

// scalarDefault returns the Python default value string for a proto scalar type.
func scalarDefault(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.BoolKind:
		return "False"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "0"
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return "0.0"
	case protoreflect.StringKind:
		return "''"
	case protoreflect.BytesKind:
		return "b''"
	case protoreflect.MessageKind:
		return "None"
	default:
		return "None"
	}
}

// writeFile writes the Python source to the generated file.
func writeFile(g *protogen.GeneratedFile, pyFile *PydanticFile) {
	opts := pyFile.Opts

	// Module docstring
	if opts.Description != "" {
		g.P(`"""`, opts.Description, `"""`)
	} else {
		g.P(`"""Generated by proto2pydantic from `, pyFile.SourceProto, `. DO NOT EDIT."""`)
	}
	g.P()
	g.P("from __future__ import annotations")
	g.P()

	// Conditional imports
	if pyFile.Imports["datetime"] || pyFile.Imports["timedelta"] {
		var dtImports []string
		if pyFile.Imports["datetime"] {
			dtImports = append(dtImports, "datetime")
		}
		if pyFile.Imports["timedelta"] {
			dtImports = append(dtImports, "timedelta")
		}
		g.P("from datetime import ", strings.Join(dtImports, ", "))
	}

	g.P("from enum import Enum")
	g.P()

	if pyFile.Imports["Any"] {
		g.P("from typing import Any")
		g.P()
	}

	// Import the base class and field_serializer if needed
	needsFieldSerializer := false
	for _, model := range pyFile.Models {
		if len(model.TimestampFields) > 0 || len(model.BytesFields) > 0 {
			needsFieldSerializer = true
			break
		}
	}

	needsBase64 := false
	for _, model := range pyFile.Models {
		if len(model.BytesFields) > 0 {
			needsBase64 = true
			break
		}
	}

	if needsBase64 {
		g.P("import base64")
	}

	baseClass := "BaseModel"
	if opts.BaseClass != "" && opts.BaseClass != "BaseModel" {
		// Custom base class: import from its module
		// e.g. "a2a._base.A2ABaseModel" -> from a2a._base import A2ABaseModel
		parts := strings.Split(opts.BaseClass, ".")
		baseClass = parts[len(parts)-1]
		modulePath := strings.Join(parts[:len(parts)-1], ".")
		g.P("from ", modulePath, " import ", baseClass)
		if needsFieldSerializer {
			g.P("from pydantic import Field, field_serializer")
		} else {
			g.P("from pydantic import Field")
		}
	} else {
		if needsFieldSerializer {
			g.P("from pydantic import BaseModel, Field, field_serializer")
		} else {
			g.P("from pydantic import BaseModel, Field")
		}
	}

	// Import alias generator dependencies
	if opts.AliasGenerator != "" {
		g.P("from pydantic import ConfigDict")
		if opts.AliasGenerator == "camel" {
			g.P("from pydantic.alias_generators import to_camel")
		}
	}

	g.P()
	g.P()

	// Write enums
	for _, enum := range pyFile.Enums {
		writeEnum(g, enum)
	}

	// Write models
	for _, model := range pyFile.Models {
		writeModel(g, model, opts)
	}

	// Write __all__ exports
	writeAllExports(g, pyFile)
}

// topologicalSort orders models so that dependencies come before dependents.
func topologicalSort(models []PydanticModel) []PydanticModel {
	// Build index: model name -> model
	index := make(map[string]int)
	for i, m := range models {
		index[m.Name] = i
	}

	// Build adjacency: model -> models it depends on
	deps := make(map[string][]string)
	for _, m := range models {
		for _, f := range m.Fields {
			// If the field type references another model in this file, it's a dependency
			baseType := strings.TrimPrefix(f.PythonType, "list[")
			baseType = strings.TrimSuffix(baseType, "]")
			baseType = strings.TrimSuffix(baseType, " | None")
			if _, ok := index[baseType]; ok && baseType != m.Name {
				deps[m.Name] = append(deps[m.Name], baseType)
			}
		}
	}

	// Kahn's algorithm for topological sort
	visited := make(map[string]bool)
	var sorted []PydanticModel
	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		for _, dep := range deps[name] {
			visit(dep)
		}
		if idx, ok := index[name]; ok {
			sorted = append(sorted, models[idx])
		}
	}

	// Visit all models preserving original order for independent models
	for _, m := range models {
		visit(m.Name)
	}

	return sorted
}

// writeAllExports writes the __all__ list at the end of the file.
func writeAllExports(g *protogen.GeneratedFile, pyFile *PydanticFile) {
	var names []string
	for _, enum := range pyFile.Enums {
		names = append(names, fmt.Sprintf("%q", enum.Name))
	}
	for _, model := range pyFile.Models {
		names = append(names, fmt.Sprintf("%q", model.Name))
	}
	if len(names) > 0 {
		g.P()
		g.P("__all__ = [")
		for _, name := range names {
			g.P("    ", name, ",")
		}
		g.P("]")
	}
}

// writeEnum writes a Python Enum class.
func writeEnum(g *protogen.GeneratedFile, enum PydanticEnum) {
	g.P("class ", enum.Name, "(str, Enum):")
	if enum.Description != "" {
		g.P(`    """`, enum.Description, `"""`)
	}
	g.P()

	if len(enum.Values) == 0 {
		g.P("    pass")
	} else {
		for _, val := range enum.Values {
			g.P("    ", val.Name, " = '", val.Value, "'")
		}
	}
	g.P()
	g.P()
}

// writeModel writes a Pydantic BaseModel class.
func writeModel(g *protogen.GeneratedFile, model PydanticModel, opts *Options) {
	// Determine base class name
	baseClass := "BaseModel"
	if opts.BaseClass != "" && opts.BaseClass != "BaseModel" {
		parts := strings.Split(opts.BaseClass, ".")
		baseClass = parts[len(parts)-1]
	}

	g.P("class ", model.Name, "(", baseClass, "):")
	if model.Description != "" {
		g.P(`    """`, model.Description, `"""`)
	}
	g.P()

	hasContent := false

	// Write model_config if alias_generator is set
	if opts.AliasGenerator == "camel" {
		g.P("    model_config = ConfigDict(")
		g.P("        populate_by_name=True,")
		g.P("        alias_generator=to_camel,")
		g.P("    )")
		g.P()
		hasContent = true
	}

	// Write regular fields
	for _, field := range model.Fields {
		writeField(g, field)
		hasContent = true
	}

	// Write oneof fields as union types
	for _, oneOf := range model.OneOfs {
		g.P("    ", oneOf.FieldName, ": ", oneOf.PythonType, " = None")
		g.P()
		hasContent = true
	}

	// Write to_proto_json() convenience method when alias_generator is set
	if opts.AliasGenerator == "camel" {
		if hasContent {
			g.P()
		}
		g.P("    def to_proto_json(self) -> dict:")
		g.P(`        """Serialize to a ProtoJSON-compatible dict (camelCase keys, no None values)."""`)
		g.P("        return self.model_dump(by_alias=True, exclude_none=True)")
		g.P()
		hasContent = true
	}

	// Write @field_serializer for Timestamp fields (datetime -> RFC 3339)
	if len(model.TimestampFields) > 0 {
		fieldList := "'" + strings.Join(model.TimestampFields, "', '") + "'"
		g.P("    @field_serializer(", fieldList, ")")
		g.P("    @classmethod")
		g.P("    def _serialize_timestamp(cls, v: datetime | None, _info: Any) -> str | None:")
		g.P(`        """Serialize datetime to RFC 3339 with UTC 'Z' suffix for ProtoJSON."""`)
		g.P("        if v is None:")
		g.P("            return None")
		g.P("        return v.strftime('%Y-%m-%dT%H:%M:%S.') + f'{v.microsecond:06d}'[:3] + 'Z'")
		g.P()
		hasContent = true
	}

	// Write @field_serializer for bytes fields (bytes -> base64)
	if len(model.BytesFields) > 0 {
		fieldList := "'" + strings.Join(model.BytesFields, "', '") + "'"
		g.P("    @field_serializer(", fieldList, ")")
		g.P("    @classmethod")
		g.P("    def _serialize_bytes(cls, v: bytes | None, _info: Any) -> str | None:")
		g.P(`        """Serialize bytes to base64 string for ProtoJSON."""`)
		g.P("        if v is None:")
		g.P("            return None")
		g.P("        return base64.b64encode(v).decode('ascii')")
		g.P()
		hasContent = true
	}

	if !hasContent {
		g.P("    pass")
	}

	g.P()
	g.P()
}

// writeField writes a single Pydantic field.
func writeField(g *protogen.GeneratedFile, field PydanticField) {
	// Build Field() arguments
	var fieldArgs []string

	// Default/required
	if field.Required {
		fieldArgs = append(fieldArgs, "...")
	} else {
		fieldArgs = append(fieldArgs, "default="+field.Default)
	}

	// buf/validate constraints
	if field.Constraints != nil && field.Constraints.HasConstraints() {
		if args := field.Constraints.ToPydanticArgs(); args != "" {
			fieldArgs = append(fieldArgs, args)
		}
	}

	// OUTPUT_ONLY -> exclude from serialization
	if field.OutputOnly {
		fieldArgs = append(fieldArgs, "exclude=True")
	}

	// Description
	if field.Description != "" {
		fieldArgs = append(fieldArgs, fmt.Sprintf("description='%s'", escapeString(field.Description)))
	}

	// If we only have a simple default and no other args, use shorthand
	hasExtras := (field.Constraints != nil && field.Constraints.HasConstraints()) ||
		field.OutputOnly || field.Description != ""

	if !field.Required && !hasExtras {
		// Simple: field_name: type = default
		g.P("    ", field.Name, ": ", field.PythonType, " = ", field.Default)
	} else {
		// Full: field_name: type = Field(..., constraint=val, description='...')
		g.P("    ", field.Name, ": ", field.PythonType, " = Field(", strings.Join(fieldArgs, ", "), ")")
	}
}

// --- Utility functions ---

// toSnakeCase converts a camelCase or PascalCase string to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				if unicode.IsLower(prev) || (i+1 < len(s) && unicode.IsLower(rune(s[i+1]))) {
					result = append(result, '_')
				}
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// enumPrefix extracts the SCREAMING_SNAKE_CASE prefix from a CamelCase enum name.
// e.g., "TaskState" -> "TASK_STATE_"
func enumPrefix(name string) string {
	var parts []string
	current := ""
	for _, r := range name {
		if unicode.IsUpper(r) && current != "" {
			parts = append(parts, strings.ToUpper(current))
			current = string(r)
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, strings.ToUpper(current))
	}
	return strings.Join(parts, "_") + "_"
}

// cleanComment removes leading slashes and whitespace from proto comments.
func cleanComment(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, " ")
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, " ")
}

// escapeString escapes single quotes in a string for Python string literals.
func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}
