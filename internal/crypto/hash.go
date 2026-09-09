package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func CalcHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
