package hasher

import (
	"crypto/sha256"
	"encoding/base64"
)

func Sha256(data, salt []byte) []byte {
	hash := sha256.New()
	hash.Write(salt)
	hash.Write(data)
	return hash.Sum(nil)
}

func Sha256String(data, salt []byte) string {
	hashsum := Sha256(data, salt)
	return base64.StdEncoding.EncodeToString(hashsum)
}
