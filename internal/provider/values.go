package provider

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Converting between Terraform values and JSON. Attribute paths are attribute names joined with
// dots, and list, set and map elements add no step, as in RENAME_NESTED.

func joinPath(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}

// fields says how attributes map to JSON: the field behind each renamed one, and how the values
// of each one the spec's unions were rewritten for translate (Transforms):
//
//   - json: a string of JSON, sent as the JSON it holds.
//   - marker: a bool, sent as an empty object when it's true.
//   - scalar_or_object: an object, sent as its value field when that's set.
type fields struct {
	renames    map[string]string
	transforms map[string]string
}

// field is the JSON field behind the attribute at path.
func (f fields) field(path, name string) string {
	if field, ok := f.renames[path]; ok {
		return field
	}
	return name
}

// toJSON converts a Terraform value to JSON. It leaves out what's null or unknown: ok is false then.
func toJSON(value tftypes.Value, path string, f fields) (out any, ok bool, err error) {
	if value.Type() == nil || value.IsNull() || !value.IsKnown() {
		return nil, false, nil
	}
	switch f.transforms[path] {
	case "json":
		var text string
		if err := value.As(&text); err != nil {
			return nil, false, err
		}
		parsed, err := parseJSON(text)
		if err != nil {
			return nil, false, fmt.Errorf("%s isn't valid JSON: %w", path, err)
		}
		return parsed, true, nil
	case "marker":
		var flag bool
		if err := value.As(&flag); err != nil || !flag {
			return nil, false, err
		}
		return map[string]any{}, true, nil
	}
	switch typ := value.Type().(type) {
	case tftypes.Object:
		attributes := elementsOf[map[string]tftypes.Value](value)
		if f.transforms[path] == "scalar_or_object" {
			if scalar, set := attributes["value"]; set && scalar.IsKnown() && !scalar.IsNull() {
				return toJSON(scalar, joinPath(path, "value"), f)
			}
		}
		object := map[string]any{}
		for name, attribute := range attributes {
			child := joinPath(path, name)
			if name == "value" && f.transforms[path] == "scalar_or_object" {
				continue
			}
			converted, ok, err := toJSON(attribute, child, f)
			if err != nil {
				return nil, false, err
			}
			if ok {
				object[f.field(child, name)] = converted
			}
		}
		return object, true, nil
	case tftypes.List, tftypes.Set, tftypes.Tuple:
		array := []any{}
		for _, element := range elementsOf[[]tftypes.Value](value) {
			converted, ok, err := toJSON(element, path, f)
			if err != nil {
				return nil, false, err
			}
			if ok {
				array = append(array, converted)
			}
		}
		return array, true, nil
	case tftypes.Map:
		object := map[string]any{}
		for key, element := range elementsOf[map[string]tftypes.Value](value) {
			converted, ok, err := toJSON(element, path, f)
			if err != nil {
				return nil, false, err
			}
			if ok {
				object[key] = converted
			}
		}
		return object, true, nil
	default:
		switch {
		case typ.Is(tftypes.String):
			var text string
			err := value.As(&text)
			return text, err == nil, err
		case typ.Is(tftypes.Bool):
			var flag bool
			err := value.As(&flag)
			return flag, err == nil, err
		case typ.Is(tftypes.Number):
			var number big.Float
			if err := value.As(&number); err != nil {
				return nil, false, err
			}
			return json.Number(number.Text('f', -1)), true, nil
		}
	}
	return nil, false, fmt.Errorf("%s: can't send a %s to the API", path, value.Type())
}

// fromJSON converts JSON to a Terraform value of type typ. When an object has no field for an
// attribute, the attribute keeps its value in prior, if prior has one: the API doesn't return
// everything it's sent, like secrets, or the path parameters it was called with.
func fromJSON(typ tftypes.Type, raw any, prior tftypes.Value, path string, f fields) (tftypes.Value, error) {
	if raw == nil {
		return tftypes.NewValue(typ, nil), nil
	}
	switch f.transforms[path] {
	case "json":
		text, err := canonicalJSON(raw)
		if err != nil {
			return tftypes.Value{}, err
		}
		// What the configuration wrote stays, if it means the same: jsonencode() and the API may
		// order fields differently.
		var written string
		if prior.Type() != nil && prior.IsKnown() && !prior.IsNull() && prior.As(&written) == nil {
			if parsed, err := parseJSON(written); err == nil {
				if same, err := canonicalJSON(parsed); err == nil && same == text {
					return prior, nil
				}
			}
		}
		return tftypes.NewValue(typ, text), nil
	case "marker":
		return tftypes.NewValue(typ, true), nil
	case "scalar_or_object":
		object, isObject := typ.(tftypes.Object)
		if !isObject {
			break
		}
		translated := map[string]any{}
		if fieldsOf, ok := raw.(map[string]any); ok {
			for name, value := range fieldsOf {
				translated[name] = value
			}
			translated["value"] = nil
		} else {
			for name := range object.AttributeTypes {
				translated[name] = nil
			}
			translated["value"] = raw
		}
		raw = translated
	}
	switch typ := typ.(type) {
	case tftypes.Object:
		object, ok := raw.(map[string]any)
		if !ok {
			return tftypes.Value{}, mismatch(path, "an object", raw)
		}
		priorAttributes := elementsOf[map[string]tftypes.Value](prior)
		attributes := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for name, attributeType := range typ.AttributeTypes {
			child := joinPath(path, name)
			field, present := object[f.field(child, name)]
			priorValue, hasPrior := priorAttributes[name]
			if !present {
				if hasPrior {
					attributes[name] = priorValue
				} else {
					attributes[name] = tftypes.NewValue(attributeType, nil)
				}
				continue
			}
			value, err := fromJSON(attributeType, field, priorValue, child, f)
			if err != nil {
				return tftypes.Value{}, err
			}
			attributes[name] = value
		}
		return tftypes.NewValue(typ, attributes), nil
	case tftypes.List, tftypes.Set:
		array, ok := raw.([]any)
		if !ok {
			return tftypes.Value{}, mismatch(path, "an array", raw)
		}
		var element tftypes.Type
		var priorElements []tftypes.Value
		if list, isList := typ.(tftypes.List); isList {
			element = list.ElementType
			priorElements = elementsOf[[]tftypes.Value](prior) // lined up by index, so nested inputs survive
		} else {
			element = typ.(tftypes.Set).ElementType
		}
		elements := make([]tftypes.Value, len(array))
		for index, item := range array {
			var priorElement tftypes.Value
			if index < len(priorElements) {
				priorElement = priorElements[index]
			}
			value, err := fromJSON(element, item, priorElement, path, f)
			if err != nil {
				return tftypes.Value{}, err
			}
			elements[index] = value
		}
		return tftypes.NewValue(typ, elements), nil
	case tftypes.Map:
		object, ok := raw.(map[string]any)
		if !ok {
			return tftypes.Value{}, mismatch(path, "an object", raw)
		}
		priorElements := elementsOf[map[string]tftypes.Value](prior)
		elements := make(map[string]tftypes.Value, len(object))
		for key, item := range object {
			value, err := fromJSON(typ.ElementType, item, priorElements[key], path, f)
			if err != nil {
				return tftypes.Value{}, err
			}
			elements[key] = value
		}
		return tftypes.NewValue(typ, elements), nil
	}

	switch {
	case typ.Is(tftypes.String):
		text, ok := scalarText(raw)
		if !ok {
			encoded, err := json.Marshal(raw)
			if err != nil {
				return tftypes.Value{}, mismatch(path, "a string", raw)
			}
			text = string(encoded)
		}
		return tftypes.NewValue(typ, text), nil
	case typ.Is(tftypes.Number):
		text, ok := scalarText(raw)
		number, _, err := big.ParseFloat(text, 10, 512, big.ToNearestEven)
		if !ok || err != nil {
			return tftypes.Value{}, mismatch(path, "a number", raw)
		}
		return tftypes.NewValue(typ, number), nil
	case typ.Is(tftypes.Bool):
		text, ok := scalarText(raw)
		flag, err := strconv.ParseBool(text)
		if !ok || err != nil {
			return tftypes.Value{}, mismatch(path, "a boolean", raw)
		}
		return tftypes.NewValue(typ, flag), nil
	}
	return tftypes.Value{}, fmt.Errorf("%s: can't read a %s from the API", path, typ)
}

// fillUnknown returns planned with what's unknown in it taken from actual. Terraform requires the
// state after an apply to keep every value the plan knew.
func fillUnknown(planned, actual tftypes.Value) tftypes.Value {
	if !planned.IsKnown() {
		return actual
	}
	if planned.IsNull() || planned.IsFullyKnown() {
		return planned
	}
	switch typ := planned.Type().(type) {
	case tftypes.Object:
		actualAttributes := elementsOf[map[string]tftypes.Value](actual)
		attributes := map[string]tftypes.Value{}
		for name, value := range elementsOf[map[string]tftypes.Value](planned) {
			other, ok := actualAttributes[name]
			if !ok {
				other = tftypes.NewValue(typ.AttributeTypes[name], nil)
			}
			attributes[name] = fillUnknown(value, other)
		}
		return tftypes.NewValue(typ, attributes)
	case tftypes.List:
		plannedElements, actualElements := elementsOf[[]tftypes.Value](planned), elementsOf[[]tftypes.Value](actual)
		if len(plannedElements) == len(actualElements) {
			elements := make([]tftypes.Value, len(plannedElements))
			for index := range plannedElements {
				elements[index] = fillUnknown(plannedElements[index], actualElements[index])
			}
			return tftypes.NewValue(typ, elements)
		}
	case tftypes.Map:
		actualElements := elementsOf[map[string]tftypes.Value](actual)
		elements := map[string]tftypes.Value{}
		for key, value := range elementsOf[map[string]tftypes.Value](planned) {
			other, ok := actualElements[key]
			if !ok {
				other = tftypes.NewValue(typ.ElementType, nil)
			}
			elements[key] = fillUnknown(value, other)
		}
		return tftypes.NewValue(typ, elements)
	}
	// A set, or a list the API returned at another length: there's nothing to line elements up by.
	if actual.IsKnown() && !actual.IsNull() {
		return actual
	}
	return unknownToNull(planned)
}

// unknownToNull replaces what's still unknown with null: after an apply, everything is known.
func unknownToNull(value tftypes.Value) tftypes.Value {
	out, err := tftypes.Transform(value, func(_ *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
		if !value.IsKnown() {
			return tftypes.NewValue(value.Type(), nil), nil
		}
		return value, nil
	})
	if err != nil {
		return value
	}
	return out
}

// withAttribute returns value with its root attribute name set to replacement.
func withAttribute(value tftypes.Value, name string, replacement tftypes.Value) tftypes.Value {
	return withAttributeAt(value, name, replacement)
}

// withAttributeAt returns value with the attribute at a dotted path set to replacement.
func withAttributeAt(value tftypes.Value, dotted string, replacement tftypes.Value) tftypes.Value {
	target := tftypes.NewAttributePath()
	for name := range strings.SplitSeq(dotted, ".") {
		target = target.WithAttributeName(name)
	}
	out, err := tftypes.Transform(value, func(at *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
		if at.Equal(target) {
			return replacement, nil
		}
		return value, nil
	})
	if err != nil {
		return value
	}
	return out
}

// attributeAt returns the value of the attribute at a dotted path.
func attributeAt(value tftypes.Value, path string) (tftypes.Value, bool) {
	for name := range strings.SplitSeq(path, ".") {
		next, ok := elementsOf[map[string]tftypes.Value](value)[name]
		if !ok {
			return tftypes.Value{}, false
		}
		value = next
	}
	return value, true
}

// attributeText returns a string, number or bool attribute as text, for a path or query parameter.
func attributeText(value tftypes.Value, path string) (string, error) {
	attribute, ok := attributeAt(value, path)
	if !ok {
		return "", fmt.Errorf("there's no attribute %s", path)
	}
	converted, set, err := toJSON(attribute, path, fields{})
	if err != nil {
		return "", err
	}
	if !set {
		return "", fmt.Errorf("%s has no value yet", path)
	}
	text, ok := scalarText(converted)
	if !ok {
		return "", fmt.Errorf("%s isn't a string, number or bool", path)
	}
	return text, nil
}

// listItems returns the items of a list response: the response itself, or the array in field.
func listItems(response any, field string) ([]any, bool) {
	if field == "" {
		items, ok := response.([]any)
		return items, ok
	}
	object, _ := response.(map[string]any)
	items, ok := object[field].([]any)
	return items, ok
}

// matchesItem reports whether the value at each JSON path in want is the text want has for it.
func matchesItem(item map[string]any, want map[string]string) bool {
	for path, text := range want {
		var value any = item
		for name := range strings.SplitSeq(path, ".") {
			object, _ := value.(map[string]any)
			value = object[name]
		}
		got, ok := scalarText(value)
		if !ok || got != text {
			return false
		}
	}
	return true
}

// parseJSON reads JSON, keeping numbers as written.
func parseJSON(text string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var parsed any
	err := decoder.Decode(&parsed)
	return parsed, err
}

// canonicalJSON writes JSON with its object fields in order, so equal values compare equal.
func canonicalJSON(value any) (string, error) {
	text, err := json.Marshal(value)
	return string(text), err
}

func scalarText(raw any) (string, bool) {
	switch value := raw.(type) {
	case string:
		return value, true
	case json.Number:
		return value.String(), true
	case bool:
		return strconv.FormatBool(value), true
	}
	return "", false
}

// elementsOf returns the attributes or elements of a known, non-null value, and the zero T otherwise.
func elementsOf[T any](value tftypes.Value) T {
	var out T
	if value.Type() == nil || value.IsNull() || !value.IsKnown() {
		return out
	}
	_ = value.As(&out)
	return out
}

func mismatch(path, want string, got any) error {
	if path == "" {
		path = "the response"
	}
	return fmt.Errorf("%s: expected %s from the API, got %T", path, want, got)
}

// isEmptyCollection reports whether value is a known, non-null list, set, tuple or map with no
// elements.
func isEmptyCollection(value tftypes.Value) bool {
	if !value.IsKnown() || value.IsNull() {
		return false
	}
	switch {
	case value.Type().Is(tftypes.List{}), value.Type().Is(tftypes.Set{}), value.Type().Is(tftypes.Tuple{}):
		var elements []tftypes.Value
		return value.As(&elements) == nil && len(elements) == 0
	case value.Type().Is(tftypes.Map{}):
		var elements map[string]tftypes.Value
		return value.As(&elements) == nil && len(elements) == 0
	}
	return false
}
