// proto2pydantic generates Pydantic v2 BaseModel classes from Protocol Buffer
// definitions, with support for google.api.field_behavior annotations.
//
// Usage as a protoc plugin:
//
//	protoc --proto2pydantic_out=. --proto2pydantic_opt=base_class=mymod.MyBase your_service.proto
//
// Usage with buf:
//
//	# buf.gen.yaml
//	plugins:
//	  - local: protoc-gen-proto2pydantic
//	    out: src/types
//	    opt:
//	      - base_class=a2a._base.A2ABaseModel
//	      - alias_generator=camel
//	      - output_file=types.py
package main

import (
	"flag"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/protocgen/proto2pydantic/generator"
)

func main() {
	var flags flag.FlagSet
	opts := &generator.Options{}

	flags.StringVar(&opts.BaseClass, "base_class", "BaseModel", "Base class for generated models (e.g. 'a2a._base.A2ABaseModel')")
	flags.StringVar(&opts.AliasGenerator, "alias_generator", "", "Alias generator for model_config ('camel' for camelCase)")
	flags.StringVar(&opts.OutputFile, "output_file", "", "Override output filename (e.g. 'types.py')")
	flags.BoolVar(&opts.StripProtoSuffix, "strip_proto_suffix", false, "Strip '_pb2_pydantic' from output filename")
	flags.StringVar(&opts.Description, "description", "", "Override module-level docstring")
	flags.StringVar(&opts.EnumStyle, "enum_style", "", "Enum style: 'raw' preserves proto names for ProtoJSON compatibility")
	flags.StringVar(&opts.Preset, "preset", "", "Preset: 'a2a' sets alias_generator=camel + enum_style=raw for ProtoJSON")

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			if err := generator.GenerateFile(gen, f, opts); err != nil {
				return err
			}
		}
		return nil
	})
}
