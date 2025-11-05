package transform

import (
	"fmt"
)

// ExampleToString demonstrates string conversion
func ExampleToString() {
	// Convert different types to string
	fmt.Println(ToString(42))
	fmt.Println(ToString(3.14))
	fmt.Println(ToString(true))
	fmt.Println(ToString("hello"))
	fmt.Println(ToString(nil))
	// Output:
	// 42 <nil>
	// 3.14 <nil>
	// true <nil>
	// hello <nil>
	//  <nil>
}

// ExampleParseInt demonstrates integer parsing
func ExampleParseInt() {
	// Parse decimal, hex, octal, and binary
	fmt.Println(ParseInt("42"))
	fmt.Println(ParseInt("0xFF"))
	fmt.Println(ParseInt("0o77"))
	fmt.Println(ParseInt("0b1010"))
	// Output:
	// 42 <nil>
	// 255 <nil>
	// 63 <nil>
	// 10 <nil>
}

// ExampleParseFloat demonstrates float parsing
func ExampleParseFloat() {
	// Parse floats with various formats
	fmt.Println(ParseFloat("3.14"))
	fmt.Println(ParseFloat("1.23e4"))
	fmt.Println(ParseFloat("-2.71"))
	// Output:
	// 3.14 <nil>
	// 12300 <nil>
	// -2.71 <nil>
}

// ExampleParseBool demonstrates boolean parsing
func ExampleParseBool() {
	// Parse various boolean representations
	fmt.Println(ParseBool("true"))
	fmt.Println(ParseBool("yes"))
	fmt.Println(ParseBool("on"))
	fmt.Println(ParseBool("false"))
	fmt.Println(ParseBool("no"))
	// Output:
	// true <nil>
	// true <nil>
	// true <nil>
	// false <nil>
	// false <nil>
}

// ExampleToInt demonstrates type conversion to int
func ExampleToInt() {
	// Convert various types to int
	fmt.Println(ToInt(42))
	fmt.Println(ToInt("100"))
	fmt.Println(ToInt(3.14))
	fmt.Println(ToInt(true))
	fmt.Println(ToInt(false))
	// Output:
	// 42 <nil>
	// 100 <nil>
	// 3 <nil>
	// 1 <nil>
	// 0 <nil>
}

// ExampleToFloat demonstrates type conversion to float
func ExampleToFloat() {
	// Convert various types to float
	fmt.Println(ToFloat(42))
	fmt.Println(ToFloat("3.14"))
	fmt.Println(ToFloat(true))
	// Output:
	// 42 <nil>
	// 3.14 <nil>
	// 1 <nil>
}

// ExampleToArray demonstrates array conversion
func ExampleToArray() {
	// Convert slices and JSON arrays to []interface{}
	arr1, _ := ToArray([]int{1, 2, 3})
	fmt.Printf("Array length: %d\n", len(arr1))

	arr2, _ := ToArray(`[1,2,3]`)
	fmt.Printf("JSON array length: %d\n", len(arr2))

	arr3, _ := ToArray("hello")
	fmt.Printf("String wrapped length: %d\n", len(arr3))
	// Output:
	// Array length: 3
	// JSON array length: 3
	// String wrapped length: 1
}

// ExampleToMap demonstrates map conversion
func ExampleToMap() {
	// Convert maps and JSON objects to map[string]interface{}
	m1, _ := ToMap(`{"name":"John","age":30}`)
	fmt.Printf("Map size: %d\n", len(m1))
	fmt.Printf("Name: %v\n", m1["name"])

	m2, _ := ToMap(map[string]string{"key": "value"})
	fmt.Printf("Map size: %d\n", len(m2))
	// Output:
	// Map size: 2
	// Name: John
	// Map size: 1
}

// ExampleIsNumeric demonstrates type checking
func ExampleIsNumeric() {
	fmt.Println(IsNumeric(42))
	fmt.Println(IsNumeric(3.14))
	fmt.Println(IsNumeric("42"))
	fmt.Println(IsNumeric(true))
	// Output:
	// true
	// true
	// false
	// false
}

// ExampleIsString demonstrates string type checking
func ExampleIsString() {
	fmt.Println(IsString("hello"))
	fmt.Println(IsString(42))
	fmt.Println(IsString(""))
	// Output:
	// true
	// false
	// true
}

// ExampleIsArray demonstrates array type checking
func ExampleIsArray() {
	fmt.Println(IsArray([]int{1, 2, 3}))
	fmt.Println(IsArray(`[1,2,3]`))
	fmt.Println(IsArray("hello"))
	fmt.Println(IsArray(42))
	// Output:
	// true
	// true
	// false
	// false
}

// ExampleIsMap demonstrates map type checking
func ExampleIsMap() {
	fmt.Println(IsMap(map[string]int{"a": 1}))
	fmt.Println(IsMap(`{"a":1}`))
	fmt.Println(IsMap("hello"))
	fmt.Println(IsMap([]int{1, 2}))
	// Output:
	// true
	// true
	// false
	// false
}

// ExampleGetType demonstrates type name retrieval
func ExampleGetType() {
	fmt.Println(GetType("hello"))
	fmt.Println(GetType(42))
	fmt.Println(GetType(3.14))
	fmt.Println(GetType([]int{1, 2}))
	fmt.Println(GetType(nil))
	// Output:
	// string
	// int
	// float64
	// []int
	// nil
}
