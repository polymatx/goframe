package util

// Contains checks if slice contains element
func Contains[T comparable](slice []T, element T) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

// Filter filters slice based on predicate
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map applies function to each element
func Map[T any, R any](slice []T, mapper func(T) R) []R {
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = mapper(item)
	}
	return result
}

// Reduce reduces slice to single value
func Reduce[T any, R any](slice []T, initial R, reducer func(R, T) R) R {
	result := initial
	for _, item := range slice {
		result = reducer(result, item)
	}
	return result
}

// Unique returns unique elements
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0)
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// Reverse reverses slice
func Reverse[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, j := 0, len(slice)-1; i < len(slice); i, j = i+1, j-1 {
		result[i] = slice[j]
	}
	return result
}

// Chunk splits slice into chunks
func Chunk[T any](slice []T, size int) [][]T {
	var chunks [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// Flatten flattens nested slices
func Flatten[T any](slices [][]T) []T {
	result := make([]T, 0)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

// First returns first element or default
func First[T any](slice []T, defaultValue T) T {
	if len(slice) == 0 {
		return defaultValue
	}
	return slice[0]
}

// Last returns last element or default
func Last[T any](slice []T, defaultValue T) T {
	if len(slice) == 0 {
		return defaultValue
	}
	return slice[len(slice)-1]
}
