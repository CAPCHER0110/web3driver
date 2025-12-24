package utils

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// VerifySignature 验证 EIP-191 格式签名
func VerifySignature(address, msg, sigHex string) bool {
	// 1. 处理签名格式
	if strings.HasPrefix(sigHex, "0x") {
		sigHex = sigHex[2:]
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != 65 {
		return false
	}

	// 2. 修正 V 值 (Ethereum 特定逻辑: 27/28 -> 0/1)
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	// 3. 构造 EIP-191 消息前缀
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(msg))
	data := []byte(prefix + msg)
	hash := crypto.Keccak256Hash(data)

	// 4. 恢复公钥
	sigPublicKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false
	}

	// 5. 导出地址并比对
	derivedAddress := crypto.PubkeyToAddress(*sigPublicKey).Hex()
	return strings.EqualFold(derivedAddress, address)
}
