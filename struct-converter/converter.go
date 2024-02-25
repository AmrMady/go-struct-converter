package struct_converter

import (
	"errors"
	"fmt"
	"reflect"
)

// convertValue recursively converts a single reflect.Value to the target type.
func convertValue(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	// Handle nil source value and zero initialization for pointers and interfaces
	if !source.IsValid() {
		return reflect.Zero(targetType), nil
	}
	// Adjust for the target being a pointer or the source being a pointer
	if targetType.Kind() == reflect.Ptr {
		// Target is a pointer type
		if source.Kind() == reflect.Ptr {
			// Source is also a pointer, dereference source for further checks
			source = source.Elem()
		}

		// Prepare a new pointer of the target type to hold the converted value
		targetPtr := reflect.New(targetType.Elem())

		// Convert the dereferenced source to the target's element type
		converted, convErr := convertValue(source, targetType.Elem())
		if convErr != nil {
			return reflect.Value{}, convErr
		}

		// Set the converted value to the newly created target pointer
		targetPtr.Elem().Set(converted)
		return targetPtr, nil
	} else if source.Kind() == reflect.Ptr {
		// Source is a pointer but target is not, dereference source and continue
		return convertValue(source.Elem(), targetType)
	}

	// Handling for non-pointer types or after adjustments for pointers
	switch source.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		// Delegate to specific conversion functions based on the kind of source
		return convertBasedOnKind(source, targetType)
	default:
		// Direct assignment or conversion for scalar and other types
		if source.Type().AssignableTo(targetType) {
			return source, nil
		} else if source.Type().ConvertibleTo(targetType) {
			return source.Convert(targetType), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("cannot convert type   %s   to   %s  ", source.Type(), targetType)
}

func convertBasedOnKind(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	// First, handle the case where source is a nil pointer.
	if source.Kind() == reflect.Ptr && source.IsNil() {
		// If the target is also a pointer type, return a nil pointer of that type.
		if targetType.Kind() == reflect.Ptr {
			return reflect.Zero(targetType), nil
		}
		// For non-pointer targetTypes, return a zero value of the targetType.
		return reflect.Zero(targetType), nil
	}

	// If the source is a pointer, dereference it for further processing.
	// This step simplifies handling by working with the value directly.
	if source.Kind() == reflect.Ptr {
		source = source.Elem()
	}

	// Determine how to convert based on the source's kind.
	switch source.Kind() {
	case reflect.Struct:
		return convertStruct(source, targetType)
	case reflect.Slice, reflect.Array:
		return convertSlice(source, targetType)
	case reflect.Map:
		return convertMap(source, targetType)
	default:
		// Handle basic type conversion and pointers specially.
		return handleBasicTypesAndPointers(source, targetType)
	}
}

func handleBasicTypesAndPointers(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	// If the target type is a pointer, we need to create a new instance of the type
	// that the pointer points to, set the value, and then return the pointer.
	if targetType.Kind() == reflect.Ptr {
		// Create a new pointer of the target type.
		targetPtr := reflect.New(targetType.Elem())

		// If the source can be directly assigned to the target, do so.
		// Otherwise, attempt to convert if the types are convertible.
		if source.Type().AssignableTo(targetType.Elem()) {
			targetPtr.Elem().Set(source)
		} else if source.Type().ConvertibleTo(targetType.Elem()) {
			targetPtr.Elem().Set(source.Convert(targetType.Elem()))
		} else {
			return reflect.Value{}, errors.New("conversion not supported")
		}

		return targetPtr, nil
	}

	// For non-pointer target types, directly assign or convert the value.
	if source.Type().AssignableTo(targetType) {
		return source, nil
	} else if source.Type().ConvertibleTo(targetType) {
		return source.Convert(targetType), nil
	}

	return reflect.Value{}, errors.New("conversion not supported")
}

func convertStruct(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	// Ensure we're dealing with structs.
	if source.Kind() == reflect.Ptr {
		source = source.Elem()
	}
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	if source.Kind() != reflect.Struct || targetType.Kind() != reflect.Struct {
		return reflect.Value{}, errors.New("source or target type is not struct or pointer to struct")
	}

	target := reflect.New(targetType).Elem()

	sourceType := source.Type()
	for i := 0; i < source.NumField(); i++ {
		sourceField := source.Field(i)
		sourceFieldName := sourceType.Field(i).Name
		targetField := target.FieldByName(sourceFieldName)

		// Skip if the target does not have a corresponding field or if it can't be set.
		if !targetField.IsValid() || !targetField.CanSet() {
			continue
		}

		// Attempt conversion based on the kind of the source field.
		var err error
		switch sourceField.Kind() {
		case reflect.Struct:
			if targetField.Kind() == reflect.Ptr {
				// Handle struct to pointer conversion
				val, convErr := convertStruct(sourceField, targetField.Type().Elem())
				if convErr == nil {
					targetField.Set(val.Addr())
				} else {
					err = convErr
				}
			} else {
				val, convErr := convertStruct(sourceField, targetField.Type())
				if convErr == nil {
					targetField.Set(val)
				} else {
					err = convErr
				}
			}
		case reflect.Slice, reflect.Array:
			convertedSlice, convErr := convertSlice(sourceField, targetField.Type())
			if convErr == nil {
				targetField.Set(convertedSlice)
			} else {
				err = convErr
			}
		case reflect.Map:
			convertedMap, convErr := convertMap(sourceField, targetField.Type())
			if convErr == nil {
				targetField.Set(convertedMap)
			} else {
				err = convErr
			}
		default:
			if sourceField.Type().AssignableTo(targetField.Type()) {
				targetField.Set(sourceField)
			} else {
				// Handle other types, possibly using convertValue for basic types or customized conversion.
				convertedValue, convErr := convertValue(sourceField, targetField.Type())
				if convErr == nil && convertedValue.IsValid() {
					targetField.Set(convertedValue)
				} else {
					err = convErr
				}
			}
		}

		if err != nil {
			return reflect.Value{}, err
		}
	}

	return target, nil
}

func convertSlice(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	if targetType.Kind() != reflect.Slice {
		return reflect.Value{}, errors.New("target type is not a slice")
	}
	elemType := targetType.Elem()
	targetSlice := reflect.MakeSlice(targetType, source.Len(), source.Cap())

	for i := 0; i < source.Len(); i++ {
		sourceElem := source.Index(i)
		convertedElem, err := convertValue(sourceElem, elemType)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to convert slice element: %v", err)
		}
		// Check if convertedElem is valid before setting it on the targetSlice.
		if convertedElem.IsValid() {
			targetSlice.Index(i).Set(convertedElem)
		} else {
			return reflect.Value{}, fmt.Errorf("converted slice element is invalid")
		}
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
	sourceVal := reflect.ValueOf(*source) //reflect.ValueOf(*source)
	targetVal := reflect.ValueOf(target).Elem()

	if sourceVal.Kind() != reflect.Struct || targetVal.Kind() != reflect.Struct {
		return errors.New("source or target is not a struct")
	}

	for i := 0; i < sourceVal.NumField(); i++ {
		sourceField := sourceVal.Field(i)
		sourceTypeField := sourceVal.Type().Field(i)
		tagValue := sourceTypeField.Tag.Get(tagName)
		if !sourceField.CanInterface() {
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
			targetField = targetVal.FieldByName(sourceTypeField.Name)
		}

		convertedVal, err := handleSetField(sourceField, targetField)
		if err != nil {
			return err
		}
		switch convertedVal.Kind() {
		case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
			// Attempt to convert the sourceField to the type expected by targetField.
			convertedVal2, err2 := convertBasedOnKind(sourceField, targetField.Type())
			if err2 != nil {
				return err2
			}

			// Ensure that convertedVal2 is compatible with targetField's type.
			// If targetField expects a pointer, ensure convertedVal2 is appropriately addressed.
			if targetField.Type().Kind() == reflect.Ptr && convertedVal2.Kind() != reflect.Ptr {
				// If convertedVal2 is not a pointer but targetField expects one, address convertedVal2.
				if convertedVal2.CanAddr() {
					targetField.Set(convertedVal2.Addr())
				} else {
					// If convertedVal2 cannot be addressed directly, create a new value and set it.
					newVal := reflect.New(convertedVal2.Type())
					newVal.Elem().Set(convertedVal2)
					targetField.Set(newVal)
				}
			} else if targetField.Type().Kind() != reflect.Ptr && convertedVal2.Kind() == reflect.Ptr {
				// If targetField does not expect a pointer but convertedVal2 is a pointer, dereference it.
				targetField.Set(convertedVal2.Elem())
			} else {
				// If the types match (both are pointers or both are not pointers), set directly.
				targetField.Set(convertedVal2)
			}
		default:
			if sourceField.Type().AssignableTo(targetField.Type()) {
				targetField.Set(sourceField)
			} else if sourceField.Type().ConvertibleTo(targetField.Type()) {
				sourceField.Convert(targetField.Type())
			}
		}
	}

	return nil
}

func handleSetField(sourceField reflect.Value, targetField reflect.Value) (reflect.Value, error) {
	if !targetField.IsValid() {
		return reflect.Value{}, nil
	}
	if sourceField.Type().AssignableTo(targetField.Type()) {
		targetField.Set(sourceField)
		return reflect.Value{}, nil
	} else {
		if !sourceField.IsValid() || sourceField.IsNil() || sourceField.IsZero() {
			return reflect.Value{}, nil
		}
		if sourceField.Kind() == reflect.Ptr {
			sourceField = sourceField.Elem()
		}
		switch sourceField.Kind() {
		case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
			// Delegate to specific conversion functions based on the kind of source
			return convertBasedOnKind(sourceField, targetField.Type())
		default:
			if sourceField.Type().AssignableTo(targetField.Type()) {
				return sourceField, nil
			} else if sourceField.Type().ConvertibleTo(targetField.Type()) {
				return sourceField.Convert(targetField.Type()), nil
			}
		}
	}

	return reflect.Value{}, errors.New("invalid_type_conversion")
}
