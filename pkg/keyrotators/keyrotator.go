package keyrotators

type KeyRotator interface {
	GetNext() string
}
