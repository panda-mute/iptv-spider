package utils

import (
	"crypto/aes"
	"crypto/md5"
)

type AESForNodejs struct {
	key [16]byte
}

func NewAESForNodejs(key []byte) *AESForNodejs {
	c := AESForNodejs{}
	c.setKey(key)
	return &c
}

func (a *AESForNodejs) setKey(key []byte) {
	a.key = md5.Sum(key)
}

func (a *AESForNodejs) Decrypt(data []byte) []byte {
	cipher, _ := aes.NewCipher(a.key[:])
	decrypted := make([]byte, len(data))

	for bs, be := 0, cipher.BlockSize(); bs < len(data); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Decrypt(decrypted[bs:be], data[bs:be])
	}

	return UnPadding(decrypted)
}

func (a *AESForNodejs) Encrypt(data []byte) []byte {
	// Java's AES/ECB/PKCS5Padding uses the AES block size (16 bytes).
	// Padding to 8 bytes could leave a partial AES block and panic.
	plain := PKCS7Padding(data)
	cipher, _ := aes.NewCipher(a.key[:])
	encrypted := make([]byte, len(plain))
	// 分组分块加密
	for bs, be := 0, cipher.BlockSize(); bs < len(plain); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Encrypt(encrypted[bs:be], plain[bs:be])
	}

	return encrypted
}
