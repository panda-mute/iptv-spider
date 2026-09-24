package model

import (
	"encoding/hex"
	"encoding/json"
	"iptv-spider/utils"
)

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

func (a *Authenticator) ToJson() []byte {
	j, _ := json.Marshal(a)
	return j
}

func (a *Authenticator) FormJson(j []byte) *Authenticator {
	json.Unmarshal(j, a)
	return a
}

func (a *Authenticator) GetEncryptString() string {
	aes := utils.NewAESForNodejs([]byte("123456"))
	return hex.EncodeToString(aes.Encrypt(a.ToJson()))
}

func NewAuthenticator(encryptToken, uid, sn, ip, mac string) Authenticator {
	return Authenticator{
		Random:       utils.GenerateRandomInt(8),
		EncryptToken: encryptToken,
		UserID:       uid,
		SN:           sn,
		IP:           utils.GenerateIP(ip, ","),
		MAC:          mac,
		MagicCode:    "CTC",
		UpdateTime:   "20230301175307", // TODO 变换
	}
}
