/*
 Copyright © 2023 MicroOps-cn.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package safe

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/tea" //nolint:staticcheck
)

var SecretEnvName = "GLOBAL_ENCRYPT_KEY"

func init() {
	key := os.Getenv(SecretEnvName)
	switch len(key) {
	default:
		fmt.Printf("Crypto: invalid key size %d, please check the environment variable `%s`.\n", len(key), SecretEnvName)
		os.Exit(1)
	case 8, 16, 24, 32:
		break
	}
}

func pkcs5Padding(data []byte, blockSize int) ([]byte, int) {
	padding := blockSize - len(data)%blockSize
	slice1 := []byte{byte(padding)}
	slice2 := bytes.Repeat(slice1, padding)
	return append(data, slice2...), padding
}

type EncryptOptions struct {
	algo    EncryptionAlgorithm
	mode    BlockMode
	padding PaddingMethod
	fixAlgo bool
}

func NewEncryptOptions(ofs ...WithOptionsFunc) *EncryptOptions {
	o := EncryptOptions{algo: Algorithm3DES, mode: BlockModeCBC, padding: PKCS5Padding, fixAlgo: false}
	for _, of := range ofs {
		of(&o)
	}
	return &o
}

type WithOptionsFunc func(o *EncryptOptions)

func WithAlgorithm(algo EncryptionAlgorithm) WithOptionsFunc {
	return func(o *EncryptOptions) {
		o.algo = algo
	}
}

func WithMode(mode BlockMode) WithOptionsFunc {
	return func(o *EncryptOptions) {
		o.mode = mode
	}
}

func WithPadding(padding PaddingMethod) WithOptionsFunc {
	return func(o *EncryptOptions) {
		o.padding = padding
	}
}

func WithFixAlgo(o *EncryptOptions) {
	o.fixAlgo = true
}

func WithoutFixAlgo(o *EncryptOptions) {
	o.fixAlgo = false
}

func fixAlgo(key string, o *EncryptOptions) {
	switch len(key) {
	case 8:
		if o.algo != AlgorithmDES {
			o.algo = AlgorithmDES
		}
	case 16:
		if o.algo != AlgorithmAES && o.algo != AlgorithmTEA {
			o.algo = AlgorithmAES
		}
	case 24:
		if o.algo != AlgorithmAES && o.algo != Algorithm3DES {
			o.algo = Algorithm3DES
		}
	case 32:
		if o.algo != AlgorithmAES {
			o.algo = AlgorithmAES
		}
	}
}

func Encrypt(originalBytes []byte, key string, o *EncryptOptions) (string, error) {
	if o == nil {
		o = NewEncryptOptions(WithFixAlgo)
	}
	if o.fixAlgo {
		fixAlgo(key, o)
	}
	block, err := o.algo.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	paddingBytes, paddingLength, err := o.padding.Padding(originalBytes, blockSize)
	if err != nil {
		return "", err
	}
	iv := make([]byte, blockSize)
	_, err = rand.Read(iv)
	if err != nil {
		return "", err
	}
	args := []byte{byte(o.mode)<<4 | byte(o.padding), 0, 0, 0, 0}
	binary.BigEndian.PutUint32(args[1:5], uint32(paddingLength))
	if len(iv) >= 5 {
		args[0] = iv[0] ^ args[0]
		args[1] = iv[1] ^ args[1]
		args[2] = iv[2] ^ args[2]
		args[3] = iv[3] ^ args[3]
		args[4] = iv[4] ^ args[4]
	}
	args = append(args, iv...)
	blockMode := o.mode.NewEncrypter(block, iv)
	cipherBytes := make([]byte, len(paddingBytes))
	blockMode.CryptBlocks(cipherBytes, paddingBytes)
	prefix := fmt.Sprintf("%s$%s$%s$", ciphertextPrefix, o.algo, base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(args))
	return prefix + base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(cipherBytes), nil
}

const ciphertextPrefix = "{CRYPT}"

func Decrypt(cipherString, key string) ([]byte, error) {
	if len(cipherString) <= len(ciphertextPrefix)+5 {
		return nil, errors.New("invalid ciphertext")
	}
	if !strings.HasPrefix(cipherString, ciphertextPrefix) ||
		cipherString[len(ciphertextPrefix)] != '$' {
		return nil, errors.New("invalid ciphertext")
	}

	buf := bytes.NewBufferString(cipherString[len(ciphertextPrefix)+1:])
	var algo EncryptionAlgorithm
	{
		if algoStr, err := buf.ReadString('$'); err != nil {
			return nil, err
		} else if len(algoStr) < 1 || len(algoStr) > 3 {
			return nil, errors.New("invalid encryption algorithm: " + strings.TrimSuffix(algoStr, "$"))
		} else {
			algoStr = algoStr[:len(algoStr)-1]
			if len(algoStr) == 1 {
				algoStr = "0" + algoStr
			}
			algos, err := hex.DecodeString(algoStr)
			if err != nil {
				return nil, err
			}
			algo = EncryptionAlgorithm(algos[0])
		}
	}
	block, err := algo.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	args, err := buf.ReadBytes('$')
	if err != nil {
		return nil, err
	}
	args = bytes.TrimSuffix(args, []byte("$"))
	args, err = base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(string(args))
	if err != nil {
		return nil, err
	}

	iv := args[5:]
	if len(iv) >= 5 {
		args[0] = iv[0] ^ args[0]
		args[1] = iv[1] ^ args[1]
		args[2] = iv[2] ^ args[2]
		args[3] = iv[3] ^ args[3]
		args[4] = iv[4] ^ args[4]
	}
	mode := BlockMode(args[0] >> 4)
	padding := PaddingMethod(args[0] & 0x0F)

	s := buf.String()
	cipherBytes, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(s)
	if err != nil {
		return nil, err
	}
	blockMode := mode.NewDecrypter(block, iv)
	paddingBytes := make([]byte, len(cipherBytes))
	blockMode.CryptBlocks(paddingBytes, cipherBytes)
	paddingLength := int(binary.BigEndian.Uint32(args[1:5]))
	originalBytes, err := padding.UnPadding(paddingBytes, paddingLength)
	if err != nil {
		return nil, err
	}
	return originalBytes, nil
}

type EncryptionAlgorithm byte

func (algo EncryptionAlgorithm) String() string {
	return strings.ToUpper(strings.TrimPrefix(hex.EncodeToString([]byte{byte(algo)}), "0"))
}

const (
	Algorithm3DES EncryptionAlgorithm = 0
	AlgorithmDES  EncryptionAlgorithm = 1
	AlgorithmAES  EncryptionAlgorithm = 2
	AlgorithmTEA  EncryptionAlgorithm = 3
)

func (algo EncryptionAlgorithm) NewCipher(key []byte) (cipher.Block, error) {
	var err error
	var block cipher.Block
	switch algo {
	case AlgorithmDES:
		block, err = des.NewCipher(key)
	case Algorithm3DES:
		block, err = des.NewTripleDESCipher(key)
	case AlgorithmAES:
		block, err = aes.NewCipher(key)
	case AlgorithmTEA:
		block, err = tea.NewCipher(key)
	default:
		return nil, fmt.Errorf("unknown encryption algorithm 0x%X", algo)
	}
	return block, err
}

type BlockMode byte

const (
	BlockModeCBC = 0x00
)

func (g BlockMode) NewEncrypter(b cipher.Block, iv []byte) cipher.BlockMode {
	switch g {
	case BlockModeCBC:
		return cipher.NewCBCEncrypter(b, iv)
	}
	return nil
}

func (g BlockMode) NewDecrypter(b cipher.Block, iv []byte) cipher.BlockMode {
	switch g {
	case BlockModeCBC:
		return cipher.NewCBCDecrypter(b, iv)
	}
	return nil
}

type PaddingMethod byte

const (
	PKCS5Padding PaddingMethod = 0x00
)

func (p PaddingMethod) Padding(data []byte, blockSize int) ([]byte, int, error) {
	switch p {
	case PKCS5Padding:
		newData, paddingLength := pkcs5Padding(data, blockSize)
		return newData, paddingLength, nil
	default:
		return nil, 0, fmt.Errorf("unknown padding 0x%X", p)
	}
}

func (p PaddingMethod) UnPadding(data []byte, paddingLength int) ([]byte, error) {
	switch p {
	case PKCS5Padding:
		return data[:(len(data) - int(paddingLength))], nil
	default:
		return nil, fmt.Errorf("unknown padding 0x%X", p)
	}
}
