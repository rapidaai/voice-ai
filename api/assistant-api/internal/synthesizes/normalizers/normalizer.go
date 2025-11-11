package internal_normalizers

type Normalizer interface {
	Normalize(text string) string
}
