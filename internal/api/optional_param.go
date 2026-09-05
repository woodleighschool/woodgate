package api

import (
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// OptionalParam preserves whether a non-pointer query parameter was supplied.
type OptionalParam[T any] struct {
	Value T
	Set   bool
}

// Schema returns the schema of the wrapped value.
func (param OptionalParam[T]) Schema(registry huma.Registry) *huma.Schema {
	return huma.SchemaFromType(registry, reflect.TypeOf(param.Value))
}

// Receiver exposes the wrapped value to Huma's parameter decoder.
func (param *OptionalParam[T]) Receiver() reflect.Value {
	return reflect.ValueOf(param).Elem().FieldByName("Value")
}

// OnParamSet records whether the request supplied the parameter.
func (param *OptionalParam[T]) OnParamSet(set bool, _ any) { param.Set = set }

// Pointer returns the wrapped value when it was supplied.
func (param OptionalParam[T]) Pointer() *T {
	if !param.Set {
		return nil
	}
	return &param.Value
}
