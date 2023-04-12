package util

func ToPtr[T any](v T) *T {
	return &v
}

func IgnoreText(input string) string {
	return "[ignore]" + input
}
