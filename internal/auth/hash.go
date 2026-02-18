package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// HashToken computes HMAC-SHA256(secret, rawToken) and returns the hex-encoded string.
// Used for verification in this module; also exported for seeds/tests that need to insert token_hash.
func HashToken(secret []byte, rawToken string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(rawToken))
	return hex.EncodeToString(mac.Sum(nil))
}
