package licensecrypto

import (
	"crypto/hmac"
	"crypto/sha256"
)

func LookupDigest(secret []byte, licenseKey string) []byte {
	n := NormalizeKey(licenseKey)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(n))
	return mac.Sum(nil)
}
