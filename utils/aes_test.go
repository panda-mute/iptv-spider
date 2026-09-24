package utils

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestAESPaddingAllBlockLengths(t *testing.T) {
	a := NewAESForNodejs([]byte("123456"))
	for n := 0; n <= 64; n++ {
		plain := bytes.Repeat([]byte{'a'}, n)
		cipher := a.Encrypt(plain)
		if len(cipher)%16 != 0 || !bytes.Equal(a.Decrypt(cipher), plain) {
			t.Fatalf("padding failed at length %d", n)
		}
	}
}

func TestAESForNodejs_Encrypt(t *testing.T) {
	data := "data12345"
	hexEnc := "58ead24a41db42bf7f9b6d53195cdf69"
	aes := NewAESForNodejs([]byte("123456"))

	resHex := hex.EncodeToString(aes.Encrypt([]byte(data)))

	if hexEnc != resHex {
		t.Fatalf("Err: result not match! -- %s", resHex)
	}
}

func TestAESForNodejs_Decrypt(t *testing.T) {
	plainText := "data12345"
	data := "58ead24a41db42bf7f9b6d53195cdf69"
	aes := NewAESForNodejs([]byte("123456"))

	b, _ := hex.DecodeString(data)

	resData := string(aes.Decrypt(b))

	if resData != plainText {
		t.Fatalf("Err: result not match! -- %s", resData)
	}

}

type Authenticator struct {
	Random       string `json:"Randon"`
	EncryptToken string `json:"EncryToken"`
	UserID       string `json:"UserID"`
	SN           string `json:"SN"`
	IP           string `json:"IP"`
	MAC          string `json:"MAC"`
	MagicCode    string `json:"MagicCode"`
	UpdateTime   string `json:"UpdateTime"`
}

func TestGenerateAuth(t *testing.T) {
	encryptToken := "f0057a9a69a842652e6f1667334f571f"
	uid := "xxxxxxxxxxxx@etv1"
	sn := "xxxxxxxxxxxx"
	mac := "xx:xx:xx:xx:xx:xx"
	ip := "22.77.xx.x"

	authInfo := Authenticator{
		Random:       GenerateRandomInt(8),
		EncryptToken: encryptToken,
		UserID:       uid,
		SN:           sn,
		IP:           GenerateIP(ip, ","),
		MAC:          mac,
		MagicCode:    "CTC",
		UpdateTime:   "20230301175307",
	}
	authJson, _ := json.Marshal(authInfo)
	authenticator := NewAESForNodejs([]byte("123456")).Encrypt(authJson)

	println(hex.EncodeToString(authenticator))
}
