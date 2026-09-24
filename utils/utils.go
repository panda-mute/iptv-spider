package utils

import (
	"crypto/md5"
	"encoding/hex"
	"go.uber.org/zap/buffer"
	"math/rand"
	"strconv"
	"strings"
)

// GenerateIP convert ip format
func GenerateIP(ip string, sep string) (result string) {
	var ipArr = strings.Split(ip, ".")
	if ip == "" {
		return
	}
	for i, s := range ipArr {
		switch len(s) {
		case 1:
			ipArr[i] = "00" + s
		case 2:
			ipArr[i] = "0" + s
		}
	}
	result = strings.Join(ipArr, sep)
	return
}

// GenerateRandomInt 生成随机数字字符串
func GenerateRandomInt(length int) (result string) {
	for i := 0; i < length; i++ {
		result += strconv.Itoa(rand.Intn(10))
	}
	return
}

// InsertStrInUserToken 插入特定字符串
// before "12345678@etv1***14670b274558c229"
// after  "123456737AE8@etv1***14670b274558c229"
func InsertStrInUserToken(token string) string {
	const index = 7
	const iStr = "37AE"
	if len(token) <= index {
		return ""
	}
	return token[:index] + iStr + token[index:]
}

// RemoveStrInUserToken 剔除特定字符串
func RemoveStrInUserToken(token string) string {
	const index = 7
	if len(token) <= index+4 {
		return ""
	}
	return token[:index] + token[index+4:]
}

func CalcMD5KeyForRequest(params ...string) string {
	hashBuf := buffer.Buffer{}
	for _, param := range params {
		if param != "" {
			hashBuf.WriteString(param)
		}
	}
	md5sum := md5.Sum(hashBuf.Bytes())
	return hex.EncodeToString(md5sum[:])
}
