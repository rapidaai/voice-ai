package ciphers

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

func Hash(str string) string {
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

func RandomHash(prefix string) string {
	uuid := uuid.New()
	return Hash(fmt.Sprintf("%s_%s", prefix, uuid.String()))
}

func Token(prefix string) string {
	sha := sha256.Sum256([]byte(RandomHash(prefix)))
	return hex.EncodeToString(sha[:])
}
