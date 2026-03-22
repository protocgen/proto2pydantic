package generator

import (
	"fmt"
	"strings"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

// FieldConstraints holds extracted buf/validate constraints for a Pydantic Field().
type FieldConstraints struct {
	MinLength        *uint64
	MaxLength        *uint64
	Pattern          string
	Email            bool
	UUID             bool
	URI              bool
	Hostname         bool
	IPv4             bool
	IPv6             bool
	Gt               string // numeric: greater than
	Gte              string // numeric: greater than or equal
	Lt               string // numeric: less than
	Lte              string // numeric: less than or equal
	MinItems         *uint64 // repeated: min items
	MaxItems         *uint64 // repeated: max items
	Const            string  // exact value constraint
	ValidateRequired bool    // (buf.validate.field).required = true
}

// HasConstraints returns true if any constraint is set.
func (c *FieldConstraints) HasConstraints() bool {
	return c.MinLength != nil || c.MaxLength != nil ||
		c.Pattern != "" || c.Email || c.UUID || c.URI || c.Hostname || c.IPv4 || c.IPv6 ||
		c.Gt != "" || c.Gte != "" || c.Lt != "" || c.Lte != "" ||
		c.MinItems != nil || c.MaxItems != nil || c.Const != "" ||
		c.ValidateRequired
}

// ToPydanticArgs returns the constraint arguments for a Pydantic Field() call.
func (c *FieldConstraints) ToPydanticArgs() string {
	var args []string

	if c.MinLength != nil {
		args = append(args, fmt.Sprintf("min_length=%d", *c.MinLength))
	}
	if c.MaxLength != nil {
		args = append(args, fmt.Sprintf("max_length=%d", *c.MaxLength))
	}
	if c.Pattern != "" {
		args = append(args, fmt.Sprintf("pattern='%s'", escapeString(c.Pattern)))
	}
	if c.Gt != "" {
		args = append(args, fmt.Sprintf("gt=%s", c.Gt))
	}
	if c.Gte != "" {
		args = append(args, fmt.Sprintf("ge=%s", c.Gte))
	}
	if c.Lt != "" {
		args = append(args, fmt.Sprintf("lt=%s", c.Lt))
	}
	if c.Lte != "" {
		args = append(args, fmt.Sprintf("le=%s", c.Lte))
	}
	if c.MinItems != nil {
		args = append(args, fmt.Sprintf("min_length=%d", *c.MinItems))
	}
	if c.MaxItems != nil {
		args = append(args, fmt.Sprintf("max_length=%d", *c.MaxItems))
	}

	return strings.Join(args, ", ")
}

// extractConstraints reads buf/validate field constraints from field options.
func extractConstraints(field *protogen.Field) *FieldConstraints {
	opts := field.Desc.Options()
	if opts == nil {
		return nil
	}

	fieldOpts, ok := opts.(*descriptorpb.FieldOptions)
	if !ok {
		return nil
	}

	if !proto.HasExtension(fieldOpts, validate.E_Field) {
		return nil
	}

	rules, ok := proto.GetExtension(fieldOpts, validate.E_Field).(*validate.FieldRules)
	if !ok || rules == nil {
		return nil
	}

	c := &FieldConstraints{}

	// Top-level required constraint
	if rules.GetRequired() {
		c.ValidateRequired = true
	}

	// String constraints
	if sr := rules.GetString_(); sr != nil {
		if sr.MinLen != nil {
			c.MinLength = sr.MinLen
		}
		if sr.MaxLen != nil {
			c.MaxLength = sr.MaxLen
		}
		if sr.Pattern != nil {
			c.Pattern = *sr.Pattern
		}
		if sr.Const != nil {
			c.Const = *sr.Const
		}
		// Well-known string formats
		if sr.GetEmail() {
			c.Email = true
		}
		if sr.GetUuid() {
			c.UUID = true
		}
		if sr.GetUri() {
			c.URI = true
		}
		if sr.GetHostname() {
			c.Hostname = true
		}
		if sr.GetIpv4() {
			c.IPv4 = true
		}
		if sr.GetIpv6() {
			c.IPv6 = true
		}
	}

	// Bytes constraints
	if br := rules.GetBytes(); br != nil {
		if br.MinLen != nil {
			c.MinLength = br.MinLen
		}
		if br.MaxLen != nil {
			c.MaxLength = br.MaxLen
		}
	}

	// Int32 constraints
	if ir := rules.GetInt32(); ir != nil {
		if v := ir.GetGt(); v != 0 {
			c.Gt = fmt.Sprintf("%d", v)
		}
		if v := ir.GetGte(); v != 0 {
			c.Gte = fmt.Sprintf("%d", v)
		}
		if v := ir.GetLt(); v != 0 {
			c.Lt = fmt.Sprintf("%d", v)
		}
		if v := ir.GetLte(); v != 0 {
			c.Lte = fmt.Sprintf("%d", v)
		}
	}

	// Int64 constraints
	if ir := rules.GetInt64(); ir != nil {
		if v := ir.GetGt(); v != 0 {
			c.Gt = fmt.Sprintf("%d", v)
		}
		if v := ir.GetGte(); v != 0 {
			c.Gte = fmt.Sprintf("%d", v)
		}
		if v := ir.GetLt(); v != 0 {
			c.Lt = fmt.Sprintf("%d", v)
		}
		if v := ir.GetLte(); v != 0 {
			c.Lte = fmt.Sprintf("%d", v)
		}
	}

	// UInt32 constraints
	if ir := rules.GetUint32(); ir != nil {
		if v := ir.GetGt(); v != 0 {
			c.Gt = fmt.Sprintf("%d", v)
		}
		if v := ir.GetGte(); v != 0 {
			c.Gte = fmt.Sprintf("%d", v)
		}
		if v := ir.GetLt(); v != 0 {
			c.Lt = fmt.Sprintf("%d", v)
		}
		if v := ir.GetLte(); v != 0 {
			c.Lte = fmt.Sprintf("%d", v)
		}
	}

	// Float constraints
	if fr := rules.GetFloat(); fr != nil {
		if v := fr.GetGt(); v != 0 {
			c.Gt = fmt.Sprintf("%g", v)
		}
		if v := fr.GetGte(); v != 0 {
			c.Gte = fmt.Sprintf("%g", v)
		}
		if v := fr.GetLt(); v != 0 {
			c.Lt = fmt.Sprintf("%g", v)
		}
		if v := fr.GetLte(); v != 0 {
			c.Lte = fmt.Sprintf("%g", v)
		}
	}

	// Double constraints
	if dr := rules.GetDouble(); dr != nil {
		if v := dr.GetGt(); v != 0 {
			c.Gt = fmt.Sprintf("%g", v)
		}
		if v := dr.GetGte(); v != 0 {
			c.Gte = fmt.Sprintf("%g", v)
		}
		if v := dr.GetLt(); v != 0 {
			c.Lt = fmt.Sprintf("%g", v)
		}
		if v := dr.GetLte(); v != 0 {
			c.Lte = fmt.Sprintf("%g", v)
		}
	}

	// Repeated constraints
	if rr := rules.GetRepeated(); rr != nil {
		if rr.MinItems != nil {
			c.MinItems = rr.MinItems
		}
		if rr.MaxItems != nil {
			c.MaxItems = rr.MaxItems
		}
	}

	if !c.HasConstraints() {
		return nil
	}
	return c
}
