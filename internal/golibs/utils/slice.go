package utils

// Contains kiểm tra một phần tử có nằm trong slice hay không
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Map chuyển đổi slice từ kiểu T sang kiểu R
func Map[T any, R any](slice []T, f func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}
	return result
}

// Filter lọc các phần tử thỏa mãn điều kiện predicate
func Filter[T any](slice []T, predicate func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// Chunk chia nhỏ slice thành các mảng con có kích thước size (rất hữu ích cho batch processing)
func Chunk[T any](slice []T, size int) [][]T {
	var chunks [][]T
	if size <= 0 {
		return chunks
	}
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}
