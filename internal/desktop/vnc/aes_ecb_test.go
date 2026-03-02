package vnc_test

import (
	"bytes"
	"crypto/aes"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/cirruslabs/mtell/internal/desktop/vnc"
	"github.com/stretchr/testify/require"
)

type ecbInput struct {
	Key       []byte
	Plaintext []byte
}

func (ecbInput) Generate(rand *rand.Rand, size int) reflect.Value {
	input := ecbInput{
		Key:       make([]byte, 16),
		Plaintext: make([]byte, rand.Intn(size+1)*aes.BlockSize),
	}

	n, err := rand.Read(input.Key)
	if err != nil {
		panic(err)
	}
	if n != len(input.Key) {
		panic("short read")
	}

	n, err = rand.Read(input.Plaintext)
	if err != nil {
		panic(err)
	}
	if n != len(input.Plaintext) {
		panic("short read")
	}

	return reflect.ValueOf(input)
}

func TestAESEncryptDecryptECB(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(input ecbInput) bool {
		ciphertext, err := vnc.AESEncryptECB(input.Plaintext, input.Key[:])
		if err != nil {
			return false
		}

		outputPlaintext, err := vnc.AESDecryptECB(ciphertext, input.Key[:])
		if err != nil {
			return false
		}

		return bytes.Equal(input.Plaintext, outputPlaintext)
	}, &quick.Config{
		MaxCountScale: 1_000,
	})
	require.NoError(t, err)
}

func TestAESEncrypDecryptRejectUnpaddedInput(t *testing.T) {
	t.Parallel()

	for i := range 100 {
		if i%aes.BlockSize == 0 {
			continue
		}

		_, err := vnc.AESEncryptECB(make([]byte, i), make([]byte, 16))
		require.Error(t, err)

		_, err = vnc.AESDecryptECB(make([]byte, i), make([]byte, 16))
		require.Error(t, err)
	}
}
