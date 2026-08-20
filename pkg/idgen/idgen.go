// Package idgen 提供 ID 与短码生成能力。
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
)

// Hex 生成 8 字节随机值的十六进制字符串（16 位）。
func Hex() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HexN 生成指定字节数的十六进制 ID。
func HexN(n int) string {
	if n <= 0 {
		n = 8
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var seq uint64

// Short 生成递增 base62 短码。
func Short() string {
	n := atomic.AddUint64(&seq, 1)
	if n == 0 {
		return string(base62Chars[0])
	}
	buf := make([]byte, 0, 11)
	for n > 0 {
		buf = append(buf, base62Chars[n%62])
		n /= 62
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
