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
	// Add cases for other complex types as necessary
	default:
		// Fallback for basic types and direct assignments
		if source.Type().AssignableTo(targetType) {
			return source, nil
		}
		// Custom basic type conversions (e.g., int to float64) can be added here
	}

	// If no suitable conversion found
	return reflect.Value{}, errors.New("incompatible types or unsupported conversion")
}

// convertStruct handles struct-to-struct conversion, respecting field names and types.
func convertStruct(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	target := reflect.New(targetType).Elem()
	for i := 0; i < source.NumField(); i++ {
		sourceField := source.Field(i)
		if !sourceField.CanInterface() {
			continue // Skip unexported fields
		}
		targetFieldName := source.Type().Field(i).Name
		targetField := target.FieldByName(targetFieldName)
		if targetField.IsValid() && targetField.CanSet() {
			convertedValue, err := convertValue(sourceField, targetField.Type())
			if err != nil {
				continue // Optionally, handle or log the error
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
			continue // Optionally, handle or log the error
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
		convertedKey, keyErr := convertValue(key, targetType.Key())
		convertedValue, valueErr := convertValue(sourceValue, targetType.Elem())
		if keyErr == nil && valueErr == nil {
			targetMap.SetMapIndex(convertedKey, convertedValue)
		}
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
		if !sourceField.CanInterface() || tagValue == "" { // Skip unexported fields
			continue
		}

		var targetField reflect.Value
		if tagName != "" {
			// Use struct tag to determine the target field name
			tagValue = sourceTypeField.Tag.Get(tagName)
			//value, ok := sourceTypeField.Tag.Lookup(tagName)
			//fmt.Println("sourceTypeField.Tag.Lookup(tagName) value: ", value, ", ok: ", ok)
			//fmt.Println("tagValue: ", tagValue)
			//fmt.Println("targetVal.FieldByName(tagValue): ", targetVal.FieldByName(tagValue))
			if tagValue != "" {
				targetField = targetVal.FieldByName(tagValue)
			}
			// If targetField is still invalid, try using the field name
			if !targetField.IsValid() || !targetField.CanSet() {
				targetField = targetVal.FieldByName(sourceTypeField.Name)
			}

			if !targetField.IsValid() || !targetField.CanSet() {
				continue // No corresponding field in the target or it's unexported
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
			continue // Optionally log the error or handle it as needed
		}

		targetField.Set(convertedValue)
	}

	return nil
}
