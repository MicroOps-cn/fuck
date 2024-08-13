package safe

import (
	"encoding/hex"
	"hash"
)

type Hash []byte

func (h Hash) HexString(n int) string {
	s := hex.EncodeToString(h)
	if len(s) > n {
		return s[:n]
	}
	return s
}

func NewHash(hashFunc func() hash.Hash, data []byte) Hash {
	h := hashFunc()
	h.Write(data)
	return h.Sum(nil)
}
