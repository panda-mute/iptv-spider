package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

const priKey = `-----BEGIN PRIVATE KEY-----
MIICdwIBADANBgkqhkiG9w0BAQEFAASCAmEwggJdAgEAAoGBAKwXs+z2tv9LDonG
jyDFWLXB9IzubHJglq/WRWMnFSb5koXu6uLpuJ6ax0MN2HG7JmDO5mT8pZn+6ayL
5HklZ43upnOCbsg3K/tFuEv4CCscztJVt/rC2y9NPJhWYfy/GxdywcAbds6mqjTY
QWbvtJv00qac1QOPprgYez4KxU6dAgMBAAECgYAmTnBigthhI1ftGyGo7cS9UJsa
88d3/kAMi+mOFJkEv/D5lyD5uYS66UEJj/9p8XqteeCXAhXqnW9uVQVaYhUWiS9l
Ois73zUpQNBT48kAMAhn7ux6gpgtSaKwSHsbKUs8YoMe0FoPKIPwln7XLhvLCKpm
5knjWD3gtVQEW8mxAQJBAOAyMMmW17fMLeSdpqskvElh63heiyb+qsmkldjYxRhK
gMK4hlI/6CFX5eIWrZUXJ1cf6UOaJFgBXQ/qxRmkSI0CQQDEgVg+jmjopestab4l
tEM3gzls4+eocwRgfzKEouo4SUtEk6BZhPcuk6Pss10Z+9jqiUMgo6TZJHYkfTWg
7kJRAkAPIlQ4x334YkgWzq2Zj/lF2t5SWc966mYNBpc29CsZ4K2gd2RZ2QaKeayC
/pTpI478SqMsdRNO/YiSsn5rpLNhAkEAttCK82/0A/VQjWhiIZvKKRwpUbfZ7qpK
uSe9LQ6QDxuJLdyWApKkkC2FBRJ9nE3kqZZX4Ea+d9HnI91lBjqDcQJBAN9v1q9e
TIM7EPiRNDlEbvJQ39+6PT8zX8EE9CHAz4mBGp4W7szMas2561xi5iN/ZC9ssUiM
jBrqDb4pgDZAFOc=
-----END PRIVATE KEY-----`

const pubKey = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCsF7Ps9rb/Sw6Jxo8gxVi1wfSM
7mxyYJav1kVjJxUm+ZKF7uri6biemsdDDdhxuyZgzuZk/KWZ/umsi+R5JWeN7qZz
gm7INyv7RbhL+AgrHM7SVbf6wtsvTTyYVmH8vxsXcsHAG3bOpqo02EFm77Sb9NKm
nNUDj6a4GHs+CsVOnQIDAQAB
-----END PUBLIC KEY-----`

func GetRSAPriKey() []byte {
	pub, _ := pem.Decode([]byte(priKey))
	return pub.Bytes
}

type RSA struct {
	pubKey *rsa.PublicKey
	priKey *rsa.PrivateKey
}

// LoadPriKey 加载私钥
func (r *RSA) LoadPriKey(key []byte) {
	priKey, err := x509.ParsePKCS8PrivateKey(key)
	if err != nil {
		priKey, _ = x509.ParsePKCS1PrivateKey(key)
	}
	r.priKey = priKey.(*rsa.PrivateKey)
}

// LoadPubKey 加载公钥
func (r *RSA) LoadPubKey(pem []byte) {
	pubKey, err := x509.ParsePKIXPublicKey(pem)
	if err != nil {
		pubKey, _ = x509.ParsePKCS1PublicKey(pem)
	}
	r.pubKey = pubKey.(*rsa.PublicKey)
}

// PubEncrypt 公钥加密
func (r *RSA) PubEncrypt(data []byte) []byte {
	result, err := rsa.EncryptPKCS1v15(rand.Reader, r.pubKey, data)
	if err != nil {
		return nil
	}
	return result
}

// PriEncrypt 私钥加密
func (r *RSA) PriEncrypt(data []byte) []byte {
	result, err := rsa.SignPKCS1v15(rand.Reader, r.priKey, crypto.Hash(0), data)
	if err != nil {
		return nil
	}
	return result
}

// PubDecrypt 公钥解密
func (r *RSA) PubDecrypt(data []byte) []byte {
	result, err := pubKeyDecrypt(r.pubKey, data)
	if err != nil {
		return nil
	}
	return result
}

// PriDecrypt 私钥解密
func (r *RSA) PriDecrypt(data []byte) []byte {
	result, err := rsa.DecryptPKCS1v15(rand.Reader, r.priKey, data)
	if err != nil {
		return nil
	}
	return result
}
