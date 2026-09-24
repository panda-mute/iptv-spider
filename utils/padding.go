package utils

import (
	"bytes"
)

func _padding(cipherText []byte, blockSize int, padInt int) []byte {
	length := (len(cipherText) + blockSize) / blockSize
	padLen := blockSize*length - len(cipherText)
	if padInt == -1 {
		padInt = padLen
	}
	padText := bytes.Repeat([]byte{byte(padInt)}, padLen)
	return append(cipherText, padText...)
}

func _unPadding(data []byte, blockSize int) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	unPaddingLen := int(data[length-1])
	// 零填充的情况
	if unPaddingLen == 0 {
		padding := data[length-blockSize:]
		for i := len(padding) - 1; i >= 0; i-- {
			if padding[i] != 0 {
				break
			}
			unPaddingLen++
		}
	}
	return data[:length-unPaddingLen]
}

func ZeroPadding(cipherText []byte, blockSize int) []byte {
	return _padding(cipherText, blockSize, 0)
}

func PKCS7Padding(cipherText []byte) []byte {
	return _padding(cipherText, 16, -1)
}

func PKCS5Padding(cipherText []byte) []byte {
	return _padding(cipherText, 8, -1)
}

func UnPadding(data []byte) []byte {
	return _unPadding(data, 16)
}
