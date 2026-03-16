package service

type HasherService interface {
	Hash(password string) (string, error)
	Compare(hashed, password string) error
}
