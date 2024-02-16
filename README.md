# Struct Converter

The `struct_converter` package provides functionality to dynamically convert between different types of structs, slices,
and maps in Go. It allows for flexible conversion of complex data structures while preserving field names and types.

## Installation

To use this package, simply import it in your Go project:

```code
import "github.com/AmrMady/go-struct-converter"
```

## Usage

####

### Example 1:

``` go
type SourceStruct struct {
    FieldA int
    FieldB string
}

type TargetStruct struct {
    A int
    B string
}

source := SourceStruct{FieldA: 42, FieldB: "example"}
var target TargetStruct
err := struct_converter.ConvertStructs(&source, &target, "")
if err != nil {
    // Handle error
}
```

####

### Example 2 (using customer struct tag in conversion):

``` go
type SourceStruct struct {
    FieldA int
    FieldB string
}

type TargetStruct struct {
    A int `custom_tag:"FieldA"`
    B string
}

source := SourceStruct{FieldA: 42, FieldB: "example"}
var target TargetStruct
err := struct_converter.ConvertStructs(&source, &target, "custom_tag")
if err != nil {
    // Handle error
}
```

####

## Additional Functions

Additional functions like `convertValue`, `convertStruct`, `convertSlice`, and `convertMap` are available for more
granular conversion of complex data structures. These functions are internally used by ConvertStructs but can be
utilized directly if needed.

####

## Contributing

Contributions are welcome! If you encounter any issues or have suggestions for improvements, please open an issue or
submit a pull request on GitHub.

####

## Contact

For any inquiries or support, feel free to contact amrsaeedmady@gmail.com.