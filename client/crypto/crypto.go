package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

// GenerateKeyPair generates a new public and private key pair for nacl/box.
func GenerateKeyPair() (string, string, error) {
	publicKey, privateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(publicKey[:]), base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

// Encrypt encrypts a message for a recipient using the sender's private key and recipient's public key.
func Encrypt(message string, recipientPublicKeyBase64 string, senderPrivateKeyBase64 string) (string, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(recipientPublicKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode recipient public key: %v", err)
	}
	var recipientPubKey [32]byte
	copy(recipientPubKey[:], pubBytes)

	privBytes, err := base64.StdEncoding.DecodeString(senderPrivateKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode sender private key: %v", err)
	}
	var senderPrivKey [32]byte
	copy(senderPrivKey[:], privBytes)

	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	encrypted := box.Seal(nonce[:], []byte(message), &nonce, &recipientPubKey, &senderPrivKey)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt decrypts a message from a sender using the recipient's private key and sender's public key.
func Decrypt(encryptedBase64 string, senderPublicKeyBase64 string, recipientPrivateKeyBase64 string) (string, error) {
	encrypted, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted message: %v", err)
	}

	if len(encrypted) < 24 {
		return "", fmt.Errorf("encrypted message too short")
	}

	var nonce [24]byte
	copy(nonce[:], encrypted[:24])

	pubBytes, err := base64.StdEncoding.DecodeString(senderPublicKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode sender public key: %v", err)
	}
	var senderPubKey [32]byte
	copy(senderPubKey[:], pubBytes)

	privBytes, err := base64.StdEncoding.DecodeString(recipientPrivateKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode recipient private key: %v", err)
	}
	var recipientPrivKey [32]byte
	copy(recipientPrivKey[:], privBytes)

	decrypted, ok := box.Open(nil, encrypted[24:], &nonce, &senderPubKey, &recipientPrivKey)
	if !ok {
		return "", fmt.Errorf("failed to decrypt message")
	}

	return string(decrypted), nil
}
