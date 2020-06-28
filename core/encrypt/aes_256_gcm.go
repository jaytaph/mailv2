package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"github.com/bitmaelum/bitmaelum-server/core/encode"
	"github.com/bitmaelum/bitmaelum-server/core/message"
	"io"
)

// Encrypt json data with AES256
func EncryptJson(key []byte, data interface{}) ([]byte, error) {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return EncryptMessage(key, plaintext)
}

// Decrypt AES256 data back into json data
func DecryptJson(key []byte, ciphertext []byte, v interface{}) error {
	plaintext, err := DecryptMessage(key, ciphertext)
	if err != nil {
		return err
	}

	return json.Unmarshal(plaintext, &v)
}

// Encrypt a binary message
func EncryptMessage(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := nonceGenerator(aead.NonceSize())
	if err != nil {
		return nil, err
	}

	return append(nonce, aead.Seal(nil, nonce, plaintext, nil)...), nil
}

// Decrypt a binary message
func DecryptMessage(key []byte, message []byte) ([]byte, error) {
	// Key should be 32byte (256bit)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	if len(message) < nonceSize {
		return nil, err
	}

	nonce, ciphertext := message[:nonceSize], message[nonceSize:]
	return aead.Open(nil, nonce, ciphertext, nil)
}

// Encrypts a catalog with a random key.
func EncryptCatalog(catalog message.Catalog) ([]byte, []byte, error) {
	catalogKey, err := keyGenerator()
	if err != nil {
		return nil, nil, err
	}

	ciphertext, err := EncryptJson(catalogKey, catalog)
	if err != nil {
		return nil, nil, err
	}

	return catalogKey, encode.Encode(ciphertext), nil
}

func DecryptCatalog(data, key []byte) (*message.Catalog, error) {
	data, err := encode.Decode(data)
	if err != nil {
		return nil, err
	}

	catalog := &message.Catalog{}
	err = DecryptJson(key, data, &catalog)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}

// Generator to generate nonces for AEAD. Used so we can easily mock it in tests
var nonceGenerator = func(size int) ([]byte, error) {
	nonce := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, nonce)

	return nonce, err
}

// Generator to generate keys for catalog encryption. Used so we can easily mock it in tests
var keyGenerator = func() ([]byte, error) {
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)

	return key, err
}
