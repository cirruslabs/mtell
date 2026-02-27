package vnc_test

import (
	"bytes"
	"crypto/md5"
	cryptorand "crypto/rand"
	"net"
	"testing"

	"github.com/monnand/dhkx"
	"github.com/stretchr/testify/require"

	"github.com/cirruslabs/mtell/internal/desktop/vnc"
)

func TestDHAuth(t *testing.T) {
	// Initialize Diffie-Hellman authentication
	auth := &vnc.DHAuth{
		Username: "alice",
		Password: "s3cr3t",
	}

	// Create a VNC connection
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	// Run Diffie-Hellman authentication
	handshakeErrCh := make(chan error, 1)
	go func() {
		handshakeErrCh <- auth.Handshake(clientConn)
	}()

	// Craft and send an authentication challenge
	group, err := dhkx.GetGroup(2)
	require.NoError(t, err)

	serverKey, err := group.GeneratePrivateKey(cryptorand.Reader)
	require.NoError(t, err)

	challenge := vnc.DHChallenge{
		Generator:       uint16(group.G().Uint64()),
		KeyLength:       uint16(len(group.P().Bytes())),
		Prime:           group.P().Bytes(),
		ServerPublicKey: serverKey.Bytes(),
	}
	require.NoError(t, challenge.Pack(serverConn))

	// Receive, parse and validate the authentication response
	var response vnc.DHResponse
	require.NoError(t, response.Unpack(serverConn, challenge.KeyLength))
	require.Len(t, response.EncryptedCredentials, 128)
	require.Len(t, response.ClientPublicKey, 128)

	clientKey := dhkx.NewPublicKey(response.ClientPublicKey)
	sharedKey, err := group.ComputeKey(clientKey, serverKey)
	require.NoError(t, err)

	sharedKeyDigest := md5.Sum(sharedKey.Bytes())
	plaintextCredentials, err := vnc.AESDecryptECB(response.EncryptedCredentials, sharedKeyDigest[:])
	require.NoError(t, err)

	require.Equal(t, auth.Username, nullTerminatedString(plaintextCredentials[:64]))
	require.Equal(t, auth.Password, nullTerminatedString(plaintextCredentials[64:]))

	// Ensure that Diffie-Hellman authentication succeeded
	require.NoError(t, <-handshakeErrCh)
}

func nullTerminatedString(block []byte) string {
	s, _, _ := bytes.Cut(block, []byte{0x00})

	return string(s)
}
