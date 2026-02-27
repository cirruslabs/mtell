package vnc

import (
	"crypto/md5"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"net"

	"github.com/monnand/dhkx"
)

const credentialBlockSize = 64

// DHAuth implements Diffie-Hellman authentication[1] for RFB protocol.
//
// This authentication is used by macOS Screen Sharing, for example.
//
// [1]: https://github.com/rfbproto/rfbproto/blob/152107db63cd34b3536ad8ddf54a0cfc9017a9f9/rfbproto.rst#diffie-hellman-authentication
type DHAuth struct {
	Username string
	Password string
}

type DHChallenge struct {
	Generator       uint16
	KeyLength       uint16
	Prime           []byte
	ServerPublicKey []byte
}

func (challenge *DHChallenge) Pack(writer io.Writer) error {
	if err := binary.Write(writer, binary.BigEndian, challenge.Generator); err != nil {
		return err
	}

	if err := binary.Write(writer, binary.BigEndian, challenge.KeyLength); err != nil {
		return err
	}

	if _, err := writer.Write(challenge.Prime); err != nil {
		return err
	}

	if _, err := writer.Write(challenge.ServerPublicKey); err != nil {
		return err
	}

	return nil
}

func (challenge *DHChallenge) Unpack(reader io.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &challenge.Generator); err != nil {
		return err
	}

	if err := binary.Read(reader, binary.BigEndian, &challenge.KeyLength); err != nil {
		return err
	}

	if challenge.KeyLength > 512 {
		return fmt.Errorf("DH key lengths larger than %d bytes are not supported",
			challenge.KeyLength)
	}

	challenge.Prime = make([]byte, challenge.KeyLength)

	if _, err := io.ReadFull(reader, challenge.Prime); err != nil {
		return err
	}

	challenge.ServerPublicKey = make([]byte, challenge.KeyLength)

	if _, err := io.ReadFull(reader, challenge.ServerPublicKey); err != nil {
		return err
	}

	return nil
}

type DHResponse struct {
	EncryptedCredentials []byte
	ClientPublicKey      []byte
}

func (response *DHResponse) Pack(writer io.Writer) error {
	if _, err := writer.Write(response.EncryptedCredentials); err != nil {
		return err
	}

	if _, err := writer.Write(response.ClientPublicKey); err != nil {
		return err
	}

	return nil
}

func (response *DHResponse) Unpack(reader io.Reader, keyLength uint16) error {
	response.EncryptedCredentials = make([]byte, credentialBlockSize*2)

	if _, err := io.ReadFull(reader, response.EncryptedCredentials); err != nil {
		return err
	}

	response.ClientPublicKey = make([]byte, keyLength)

	if _, err := io.ReadFull(reader, response.ClientPublicKey); err != nil {
		return err
	}

	return nil
}

func (auth *DHAuth) SecurityType() uint8 {
	return 30
}

func (auth *DHAuth) Handshake(conn net.Conn) error {
	// Read authentication challenge
	var challenge DHChallenge

	if err := challenge.Unpack(conn); err != nil {
		return err
	}

	// Initialize a Diffie-Hellman group and generate our key
	group := dhkx.CreateGroup(new(big.Int).SetBytes(challenge.Prime),
		big.NewInt(int64(challenge.Generator)))

	clientKey, err := group.GeneratePrivateKey(cryptorand.Reader)
	if err != nil {
		return err
	}

	// Compute a shared key
	serverKey := dhkx.NewPublicKey(challenge.ServerPublicKey)

	sharedKey, err := group.ComputeKey(serverKey, clientKey)
	if err != nil {
		return err
	}

	// Craft authentication response
	usernameBlock, err := paddedCredentialBlock(auth.Username)
	if err != nil {
		return err
	}
	passwordBlock, err := paddedCredentialBlock(auth.Password)
	if err != nil {
		return err
	}

	plaintextCredentials := append(usernameBlock, passwordBlock...)
	sharedKeyDigest := md5.Sum(sharedKey.Bytes())

	ciphertextCredentials, err := AESEncryptECB(plaintextCredentials, sharedKeyDigest[:])
	if err != nil {
		return err
	}

	// Send authentication response
	response := DHResponse{
		EncryptedCredentials: ciphertextCredentials,
		ClientPublicKey:      clientKey.Bytes(),
	}

	if err := response.Pack(conn); err != nil {
		return err
	}

	return nil
}

func paddedCredentialBlock(credential string) ([]byte, error) {
	const terminator = 1

	if len(credential)+terminator > credentialBlockSize {
		return nil, fmt.Errorf("credential cannot be larger than %d bytes", credentialBlockSize-terminator)
	}

	result := make([]byte, credentialBlockSize)

	n := copy(result, credential)

	if _, err := io.ReadFull(cryptorand.Reader, result[n+terminator:]); err != nil {
		return nil, err
	}

	return result, nil
}
