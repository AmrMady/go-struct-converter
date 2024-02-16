package struct_converter

import (
	"errors"
	"reflect"
)

// convertValue recursively converts a single reflect.Value to the target type.
func convertValue(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	switch source.Kind() {
	case reflect.Struct:
		return convertStruct(source, targetType)
	case reflect.Slice:
		return convertSlice(source, targetType)
	case reflect.Map:
		return convertMap(source, targetType)
	default:
		if source.Type().AssignableTo(targetType) {
			return source, nil
		}
	}

	return reflect.Value{}, errors.New("incompatible types or unsupported conversion")
}

// convertStruct handles struct-to-struct conversion, respecting field names and types.
func convertStruct(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	target := reflect.New(targetType).Elem()
	for i := 0; i < source.NumField(); i++ {
		sourceField := source.Field(i)
		if !sourceField.CanInterface() {
			continue
		}
		targetFieldName := source.Type().Field(i).Name
		targetField := target.FieldByName(targetFieldName)
		if targetField.IsValid() && targetField.CanSet() {
			convertedValue, err := convertValue(sourceField, targetField.Type())
			if err != nil {
				return reflect.Value{}, err
			}
			targetField.Set(convertedValue)
		}
	}
	return target, nil
}

// convertSlice handles slice-to-slice conversion, element by element.
func convertSlice(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	if targetType.Kind() != reflect.Slice {
		return reflect.Value{}, errors.New("target type is not a slice")
	}
	elemType := targetType.Elem()
	targetSlice := reflect.MakeSlice(targetType, source.Len(), source.Cap())
	for i := 0; i < source.Len(); i++ {
		convertedValue, err := convertValue(source.Index(i), elemType)
		if err != nil {
			return reflect.Value{}, err
		}
		targetSlice.Index(i).Set(convertedValue)
	}
	return targetSlice, nil
}

// convertMap handles map-to-map conversion, key by key and value by value.
func convertMap(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	if targetType.Kind() != reflect.Map {
		return reflect.Value{}, errors.New("target type is not a map")
	}
	targetMap := reflect.MakeMapWithSize(targetType, source.Len())
	for _, key := range source.MapKeys() {
		sourceValue := source.MapIndex(key)
		convertedKey, err := convertValue(key, targetType.Key())
		if err != nil {
			return reflect.Value{}, err
		}
		convertedValue, err := convertValue(sourceValue, targetType.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		targetMap.SetMapIndex(convertedKey, convertedValue)
	}
	return targetMap, nil
}

// ConvertStructs dynamically converts fields from a source struct to a target struct using pointers.
// It uses struct field names for matching by default but can also use a specified struct tag for matching.
func ConvertStructs[Source any, Target any](source *Source, target *Target, tagName string) error {
	sourceVal := reflect.ValueOf(*source)
	targetVal := reflect.ValueOf(target).Elem()

	if sourceVal.Kind() != reflect.Struct || targetVal.Kind() != reflect.Struct {
		return errors.New("source or target is not a struct")
	}

	for i := 0; i < sourceVal.NumField(); i++ {
		sourceField := sourceVal.Field(i)
		sourceTypeField := sourceVal.Type().Field(i)
		tagValue := sourceTypeField.Tag.Get(tagName)
		if !sourceField.CanInterface() || tagValue == "" {
			continue
		}

		var targetField reflect.Value
		if tagName != "" {
			tagValue = sourceTypeField.Tag.Get(tagName)
			if tagValue != "" {
				targetField = targetVal.FieldByName(tagValue)
			}
			if !targetField.IsValid() || !targetField.CanSet() {
				targetField = targetVal.FieldByName(sourceTypeField.Name)
			}

			if !targetField.IsValid() || !targetField.CanSet() {
				continue
			}
		} else {
			// Fallback to using the source field name if no tag is specified or the tag is empty
			targetField = targetVal.FieldByName(sourceTypeField.Name)
		}

		if !targetField.IsValid() || !targetField.CanSet() {
			continue // No corresponding field in the target, or it's unexported
		}

		convertedValue, err := convertValue(sourceField, targetField.Type())
		if err != nil {
			return err
		}

		targetField.Set(convertedValue)
	}

	return nil
}
