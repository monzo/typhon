package stringsvc

type Service interface {
	Uppercase(s string) (string, error)
	Count(s string) (int, error)
}
