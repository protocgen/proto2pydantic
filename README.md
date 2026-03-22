# proto2pydantic

A `protoc` plugin that generates [Pydantic v2](https://docs.pydantic.dev/) `BaseModel` classes from Protocol Buffer definitions — with **`google.api.field_behavior`** support.

## Why this exists

Proto3 has no native `required` keyword. Every field silently accepts zero values, which means proto-generated models can't enforce that critical fields like `message_id` or `url` are actually provided.

Google's solution is [`google.api.field_behavior`](https://google.aip.dev/203) — annotations that mark fields as `REQUIRED`, `OPTIONAL`, `OUTPUT_ONLY`, etc:

```protobuf
string url = 1 [(google.api.field_behavior) = REQUIRED];  // must be provided
string tenant = 2;                                         // optional, zero-value OK
optional int32 history_length = 3;                         // explicitly optional
```

**The problem**: no existing proto-to-Pydantic tool reads these annotations.

| Tool | Reads `field_behavior`? |
|---|:-:|
| [`protobuf-to-pydantic`](https://github.com/so1n/protobuf_to_pydantic) | ❌ Uses PGV or custom `p2p:` comments |
| [`protoc-gen-pydantic`](https://pkg.go.dev/github.com/mercurial-harsh/protoc-gen-pydantic) | ❌ Uses `optional` keyword only |
| [`protobuf-pydantic-gen`](https://github.com/danielgtaylor/python-betterproto) | ❌ Basic mapping only |
| [`python-betterproto`](https://github.com/danielgtaylor/python-betterproto) | ❌ Uses `optional` keyword only |
| **proto2pydantic** | ✅ |

This means projects like [A2A](https://github.com/a2aproject/A2A) that use `field_behavior` annotations extensively have had to go through an intermediate JSON Schema step to get proper validation in Pydantic. proto2pydantic eliminates that indirection.

## Features

- 🔒 **`field_behavior` support** — `REQUIRED` → required Pydantic field, `OUTPUT_ONLY` → `exclude=True`
- ✅ **`buf/validate` support** — proto validation rules → Pydantic `Field()` constraints
- 🐍 **Idiomatic Python** — snake_case fields, `str` Enums, `oneof` → union types
- 📦 **Well-known types** — `Struct` → `dict[str, Any]`, `Timestamp` → `datetime`
- 🔌 **buf native** — works as a local or remote buf plugin
- ⚙️ **Configurable** — custom base class, camelCase aliases, output filename

## Install

```bash
go install github.com/protocgen/proto2pydantic@latest
```

## Usage

### With buf

```yaml
# buf.gen.yaml
version: v2
plugins:
  - local: protoc-gen-proto2pydantic
    out: src/types
    opt:
      - base_class=a2a._base.A2ABaseModel
      - alias_generator=camel
      - output_file=types.py
```

```bash
buf generate
```

### With protoc

```bash
protoc --proto2pydantic_out=./output \
       --proto2pydantic_opt=base_class=myapp.BaseModel \
       your_service.proto
```

## How it works

```
.proto file → protoc/buf → proto2pydantic → .py with Pydantic models
```

Given:
```protobuf
message AgentInterface {
  string url = 1 [(google.api.field_behavior) = REQUIRED];
  string protocol_binding = 2 [(google.api.field_behavior) = REQUIRED];
  string tenant = 3;
  string protocol_version = 4 [(google.api.field_behavior) = REQUIRED];
}
```

Generates:
```python
class AgentInterface(BaseModel):
    url: str = Field(..., description='The URL where this interface is available.')
    protocol_binding: str = Field(..., description='The protocol binding supported at this URL.')
    tenant: str = Field(default='', description='Tenant ID.')
    protocol_version: str = Field(..., description='The version of the A2A protocol.')
```

- `Field(...)` = **required** — Pydantic raises `ValidationError` if missing
- `Field(default='')` = proto3 zero-value default — field is optional

## Validation with `buf/validate`

proto2pydantic reads [`buf/validate`](https://github.com/bufbuild/protovalidate) (the successor to protoc-gen-validate) and maps constraints to Pydantic `Field()` arguments:

```protobuf
import "buf/validate/validate.proto";

message CreateUserRequest {
  string email = 1 [
    (google.api.field_behavior) = REQUIRED,
    (buf.validate.field).string.email = true
  ];
  string name = 2 [
    (buf.validate.field).string = {min_len: 1, max_len: 100}
  ];
  int32 age = 3 [
    (buf.validate.field).int32 = {gte: 0, lte: 150}
  ];
  repeated string tags = 4 [
    (buf.validate.field).repeated = {min_items: 1, max_items: 10}
  ];
}
```

Generates:
```python
class CreateUserRequest(BaseModel):
    email: str = Field(...)
    name: str = Field(default='', min_length=1, max_length=100)
    age: int = Field(default=0, ge=0, le=150)
    tags: list[str] | None = Field(default=None, min_length=1, max_length=10)
```

| `buf/validate` rule | Pydantic `Field()` |
|---|---|
| `required` | `Field(...)` — no default, field is required |
| `string.min_len` | `min_length=` |
| `string.max_len` | `max_length=` |
| `string.pattern` | `pattern=` |
| `int32.gte` / `float.gte` | `ge=` |
| `int32.lte` / `float.lte` | `le=` |
| `int32.gt` / `float.gt` | `gt=` |
| `int32.lt` / `float.lt` | `lt=` |
| `repeated.min_items` | `min_length=` |
| `repeated.max_items` | `max_length=` |

## `field_behavior` annotations

| Annotation | Effect |
|---|---|
| `REQUIRED` | No default value → Pydantic requires the field |
| `OUTPUT_ONLY` | `Field(exclude=True)` → excluded from `model_dump()` |
| `OPTIONAL` | Treated as proto3 default (zero-value) |
| _(none)_ | proto3 zero-value default |

## Options

| Option | Description | Example |
|---|---|---|
| `base_class` | Custom base class for models | `a2a._base.A2ABaseModel` |
| `alias_generator` | Add `model_config` with alias generator | `camel` |
| `output_file` | Override output filename | `types.py` |
| `strip_proto_suffix` | Use `foo.py` instead of `foo_pb2_pydantic.py` | `true` |

## Type Mapping

| Proto | Python |
|---|---|
| `string` | `str` |
| `int32`, `int64`, etc. | `int` |
| `float`, `double` | `float` |
| `bool` | `bool` |
| `bytes` | `bytes` |
| `repeated T` | `list[T]` |
| `map<K, V>` | `dict[K, V]` |
| `optional T` | `T \| None` |
| `oneof` | `T1 \| T2 \| ... \| None` |
| `google.protobuf.Struct` | `dict[str, Any]` |
| `google.protobuf.Timestamp` | `datetime` |
| `google.protobuf.Value` | `Any` |
| Enum | `str` Enum (prefix-stripped, lowercase) |

## License

Apache-2.0
