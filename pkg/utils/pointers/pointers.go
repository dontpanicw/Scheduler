package pointers

func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
