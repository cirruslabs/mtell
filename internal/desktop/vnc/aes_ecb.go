package vnc

import (
	"crypto/aes"
	"fmt"
)

func AESEncryptECB(plaintext []byte, key []byte) ([]byte, error) {
	return ecb(plaintext, key, true)
}

func AESDecryptECB(plaintext []byte, key []byte) ([]byte, error) {
	return ecb(plaintext, key, false)
}

func ecb(plaintext []byte, key []byte, encrypt bool) ([]byte, error) {
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := cipher.BlockSize()

	if len(plaintext)%blockSize != 0 {
		return nil, fmt.Errorf("plaintext size should be a multiple of the cipher block size")
	}

	ciphertext := make([]byte, len(plaintext))

	for i := 0; i < len(plaintext); i += blockSize {
		src := plaintext[i : i+blockSize]
		dst := ciphertext[i : i+blockSize]

		if encrypt {
			cipher.Encrypt(dst, src)
		} else {
			cipher.Decrypt(dst, src)
		}

	}

	return ciphertext, nil
}
