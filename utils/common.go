package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateNonce 生成安全的随机字符串
func GenerateNonce() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
