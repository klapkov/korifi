package tools

import (
	"cmp"
	"reflect"
	"slices"
)

func Uniq[S ~[]E, E cmp.Ordered](elements S) S {
	slices.Sort(elements)
	return slices.Compact(elements)
}

func EmptyOrContains[S ~[]E, E comparable](elements S, e E) bool {
	if len(elements) == 0 {
		return true
	}

	return slices.Contains(elements, e)
}

func EmptyOrContainsAll[K comparable, V any](subset []K, m map[K]V, condition func(V) bool) bool {
	if len(subset) == 0 {
		return true
	}
	for _, key := range subset {
		value, exists := m[key]
		if !exists || !condition(value) {
			return false
		}
	}
	return true
}

func NilOrEquals[E comparable](value *E, expectedValue E) bool {
	if value == nil {
		return true
	}

	return expectedValue == *value
}

func ZeroOrEquals[E comparable](value E, expectedValue E) bool {
	if reflect.ValueOf(value).IsZero() {
		return true
	}

	return value == expectedValue
}

func SetMapValue[K comparable, V any](m map[K]V, key K, value V) map[K]V {
	if m == nil {
		m = map[K]V{}
	}
	m[key] = value

	return m
}

func GetMapValue[K comparable, V any](m map[K]V, key K, defaultValue V) V {
	if value, found := m[key]; found {
		return value
	}
	return defaultValue
}
