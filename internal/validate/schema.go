package validate

import (
	"reflect"

	"github.com/casualjim/go-swagger/spec"
	"github.com/casualjim/go-swagger/strfmt"
	"github.com/casualjim/go-swagger/util"
)

var specSchemaType = reflect.TypeOf(&spec.Schema{})

// SchemaValidator like param validator but for a full json schema
type SchemaValidator struct {
	Path         string
	in           string
	Schema       *spec.Schema
	validators   []valueValidator
	Root         interface{}
	KnownFormats strfmt.Registry
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(schema *spec.Schema, rootSchema interface{}, root string, formats strfmt.Registry) *SchemaValidator {
	if schema == nil {
		return nil
	}

	if rootSchema == nil {
		rootSchema = schema
	}

	if schema.ID != "" || schema.Ref.String() != "" || schema.Ref.IsRoot() {
		err := spec.ExpandSchema(schema, rootSchema, nil)
		if err != nil {
			panic(err)
		}
	}

	s := SchemaValidator{Path: root, in: "body", Schema: schema, Root: rootSchema, KnownFormats: formats}
	s.validators = []valueValidator{
		s.typeValidator(),
		s.schemaPropsValidator(),
		s.stringValidator(),
		s.formatValidator(),
		s.numberValidator(),
		s.sliceValidator(),
		s.commonValidator(),
		s.objectValidator(),
	}
	return &s
}

// SetPath sets the path for this schema valdiator
func (s *SchemaValidator) SetPath(path string) {
	s.Path = path
}

// Applies returns true when this schema validator applies
func (s *SchemaValidator) Applies(source interface{}, kind reflect.Kind) bool {
	_, ok := source.(*spec.Schema)
	return ok
}

// Validate validates the data against the schema
func (s *SchemaValidator) Validate(data interface{}) *Result {
	if data == nil {
		v := s.validators[0].Validate(data)
		v.Merge(s.validators[6].Validate(data))
		return v
	}
	result := new(Result)

	tpe := reflect.TypeOf(data)
	kind := tpe.Kind()
	for kind == reflect.Ptr {
		tpe = tpe.Elem()
		kind = tpe.Kind()
	}
	d := data
	if kind == reflect.Struct {
		d = util.ToDynamicJSON(data)
	}

	for _, v := range s.validators {
		if !v.Applies(s.Schema, kind) {
			continue
		}

		err := v.Validate(d)
		result.Merge(err)
		result.Inc()
	}
	result.Inc()
	return result
}

func (s *SchemaValidator) typeValidator() valueValidator {
	return &typeValidator{Type: s.Schema.Type, Format: s.Schema.Format, In: s.in, Path: s.Path}
}

func (s *SchemaValidator) commonValidator() valueValidator {
	return &basicCommonValidator{
		Path:    s.Path,
		In:      s.in,
		Default: s.Schema.Default,
		Enum:    s.Schema.Enum,
	}
}

func (s *SchemaValidator) sliceValidator() valueValidator {
	return &schemaSliceValidator{
		Path:            s.Path,
		In:              s.in,
		MaxItems:        s.Schema.MaxItems,
		MinItems:        s.Schema.MinItems,
		UniqueItems:     s.Schema.UniqueItems,
		AdditionalItems: s.Schema.AdditionalItems,
		Items:           s.Schema.Items,
		Root:            s.Root,
		KnownFormats:    s.KnownFormats,
	}
}

func (s *SchemaValidator) numberValidator() valueValidator {
	return &numberValidator{
		Path:             s.Path,
		In:               s.in,
		Default:          s.Schema.Default,
		MultipleOf:       s.Schema.MultipleOf,
		Maximum:          s.Schema.Maximum,
		ExclusiveMaximum: s.Schema.ExclusiveMaximum,
		Minimum:          s.Schema.Minimum,
		ExclusiveMinimum: s.Schema.ExclusiveMinimum,
	}
}

func (s *SchemaValidator) stringValidator() valueValidator {
	return &stringValidator{
		Path:      s.Path,
		In:        s.in,
		Default:   s.Schema.Default,
		MaxLength: s.Schema.MaxLength,
		MinLength: s.Schema.MinLength,
		Pattern:   s.Schema.Pattern,
	}
}

func (s *SchemaValidator) formatValidator() valueValidator {
	return &formatValidator{
		Path:         s.Path,
		In:           s.in,
		Default:      s.Schema.Default,
		Format:       s.Schema.Format,
		KnownFormats: s.KnownFormats,
	}
}

func (s *SchemaValidator) schemaPropsValidator() valueValidator {
	sch := s.Schema
	return newSchemaPropsValidator(s.Path, s.in, sch.AllOf, sch.OneOf, sch.AnyOf, sch.Not, sch.Dependencies, s.Root, s.KnownFormats)
}

func (s *SchemaValidator) objectValidator() valueValidator {
	return &objectValidator{
		Path:                 s.Path,
		In:                   s.in,
		MaxProperties:        s.Schema.MaxProperties,
		MinProperties:        s.Schema.MinProperties,
		Required:             s.Schema.Required,
		Properties:           s.Schema.Properties,
		AdditionalProperties: s.Schema.AdditionalProperties,
		PatternProperties:    s.Schema.PatternProperties,
		Root:                 s.Root,
		KnownFormats:         s.KnownFormats,
	}
}
