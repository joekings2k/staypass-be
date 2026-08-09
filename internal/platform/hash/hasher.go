package hash

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}
