# proto2pydantic (Archived)

> **This project has been archived.** All functionality has been absorbed into [proto2type](https://github.com/protocgen/proto2type).

## Migration Guide

Replace your `buf.gen.yaml` plugin entry:

### Before (proto2pydantic)

```yaml
version: v2
plugins:
  - local: protoc-gen-proto2pydantic
    out: gen/python
    opt:
      - alias_generator=camel
      - enum_style=raw
```

### After (proto2type)

```yaml
version: v2
plugins:
  - local: protoc-gen-proto2type
    out: gen/python
    opt:
      - lang=python
      - preset=a2a
```

Or with explicit options:

```yaml
version: v2
plugins:
  - local: protoc-gen-proto2type
    out: gen/python
    opt:
      - lang=python
      - alias_generator=camel
      - enum_style=raw
```

## What changed?

- All proto2pydantic features are now available in proto2type's Python backend (`lang=python`)
- The `preset=a2a` option replaces the common `alias_generator=camel` + `enum_style=raw` combination
- `google.api.field_behavior` and `buf/validate` constraints are fully supported
- Bug fixes: `gte: 0` constraints no longer silently dropped, enum defaults no longer reference skipped `_UNSPECIFIED` values
- proto2type also supports Go, Rust, and Kotlin backends from the same proto definitions

## Installation

Install proto2type instead:

```bash
go install github.com/protocgen/proto2type@latest
```

See the [proto2type README](https://github.com/protocgen/proto2type) for full documentation.
