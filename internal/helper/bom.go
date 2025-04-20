package helper

import (
	"bytes"
	"encoding/json"
	"iter"
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"unicode"
)

// [min, max]
func RandRange(min, max int) int {
	return rand.Intn(max+1-min) + min
}

func Map[T, U any](arr []T, mapper func(t T) U) []U {
	result := make([]U, len(arr))

	for i, elem := range arr {
		result[i] = mapper(elem)
	}

	return result
}

func Zip[T, U any](t []T, u []U) iter.Seq2[T, U] {
	return func(yield func(T, U) bool) {
		for i := range min(len(t), len(u)) {
			if !yield(t[i], u[i]) {
				return
			}
		}
	}
}

func ParseAsJson(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

func Filter[T any](arr []T, pred func(T) bool) []T {
	var res []T
	for _, v := range arr {
		if pred(v) {
			res = append(res, v)
		}
	}
	return res
}

func StripWS(s string) string {
	s = strings.TrimSpace(s)

	var b strings.Builder
	b.Grow(len(s))

	isSpace := false
	for _, ch := range s {
		if !unicode.IsSpace(ch) {
			b.WriteRune(ch)
			isSpace = false
		} else if !isSpace {
			b.WriteRune(' ')
			isSpace = true
		}
	}

	return b.String()
}

func Contains[T comparable](arr []T, elem T) bool {
	return slices.Contains(arr, elem)
}

func GetStructFieldsInSnakeCase[T any](obj T) []string {
	fields := reflect.Indirect(reflect.ValueOf(obj)).Type().NumField()
	result := make([]string, fields)

	for i := range fields {
		field := reflect.Indirect(reflect.ValueOf(obj)).Type().Field(i).Name
		result[i] = ConvertCamelToSnake(field)
	}

	return result
}

func ConvertCamelToSnake(name string) string {
	var buffer bytes.Buffer

	for i, c := range name {
		if unicode.IsUpper(c) {
			if i != 0 {
				buffer.WriteRune('_')
			}
			buffer.WriteRune(unicode.ToLower(c))
		} else {
			buffer.WriteRune(c)
		}
	}

	return buffer.String()
}
