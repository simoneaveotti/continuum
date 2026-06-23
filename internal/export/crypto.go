package export

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"continuum/internal/prompt"

	"golang.org/x/crypto/argon2"
)

func encryptData(data []byte, passphrase string, algo EncryptionAlgo) ([]byte, error) {
	algo = algo.Default()

	switch algo {
	case AlgoAES_GCM_V2, "":
		return encryptAESGCMV2(data, passphrase)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
}

func decryptData(data []byte, passphrase string, algo EncryptionAlgo) ([]byte, error) {
	switch algo {
	case "", AlgoAES_GCM_V2:
		return decryptAESGCMV2(data, passphrase)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
}

func encryptAESGCMV2(data []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, v2SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := deriveKeyArgon2(passphrase, salt, v2ArgonTime, v2ArgonMemory, v2ArgonThreads)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)
	buf := bytes.NewBuffer(make([]byte, 0, len(v2Magic)+13+len(salt)+len(nonce)+len(ciphertext)))
	buf.WriteString(v2Magic)
	_ = binary.Write(buf, binary.BigEndian, v2ArgonTime)
	_ = binary.Write(buf, binary.BigEndian, v2ArgonMemory)
	buf.WriteByte(v2ArgonThreads)
	buf.WriteByte(byte(len(salt)))
	buf.WriteByte(byte(len(nonce)))
	buf.Write(salt)
	buf.Write(nonce)
	buf.Write(ciphertext)
	return buf.Bytes(), nil
}

func decryptAESGCMV2(data []byte, passphrase string) ([]byte, error) {
	if !strings.HasPrefix(string(data), v2Magic) {
		return nil, fmt.Errorf("invalid %s payload", AlgoAES_GCM_V2)
	}
	reader := bytes.NewReader(data[len(v2Magic):])

	var timeCost uint32
	var memoryCost uint32
	var threads uint8
	var saltLen uint8
	var nonceLen uint8

	if err := binary.Read(reader, binary.BigEndian, &timeCost); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &memoryCost); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &threads); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &saltLen); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &nonceLen); err != nil {
		return nil, fmt.Errorf("invalid v2 header: %w", err)
	}
	if saltLen == 0 || nonceLen == 0 {
		return nil, fmt.Errorf("invalid v2 header lengths")
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(reader, salt); err != nil {
		return nil, fmt.Errorf("invalid v2 salt: %w", err)
	}
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(reader, nonce); err != nil {
		return nil, fmt.Errorf("invalid v2 nonce: %w", err)
	}
	ciphertext, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	key := deriveKeyArgon2(passphrase, salt, timeCost, memoryCost, threads)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func deriveKeyArgon2(passphrase string, salt []byte, timeCost, memoryCost uint32, threads uint8) []byte {
	return argon2.IDKey([]byte(passphrase), salt, timeCost, memoryCost, threads, v2KeySize)
}

func promptPassphrase() (string, error) {
	return prompt.ReadLine("Enter passphrase: ")
}

func promptDecryptPassphrase() (string, error) {
	return prompt.ReadLine("Enter decryption passphrase: ")
}
