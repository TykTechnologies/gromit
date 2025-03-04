package util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// MD5Sum returns "" if there is a problem computing the sum
func Md5Sum(f *os.File) string {
	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(nil))
}
