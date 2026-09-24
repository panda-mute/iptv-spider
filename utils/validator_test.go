package utils

import "testing"

func TestCheckMacAddressV1(t *testing.T) {
	data := map[string]bool{
		"01:02:03:04:ab:cd":    true,
		"definitely:not:a:mac": false,
		"01-02-03-04-ab-cd":    false,
	}

	for s, b := range data {
		r := CheckMacAddressV1(s)
		if r != b {
			t.Fatalf("错误: 数据 -- %s, 期待结果 %t, 计算结果 %t", s, b, r)
		}
	}
}

func TestCheckIPv4Address(t *testing.T) {
	data := map[string]bool{
		"127.0.0.1":         true,
		"0.0.0.0":           true,
		"255.255.255.255.1": false,
		"256.256.256.256":   false,
		"999.999.999.999":   false,
		"1.2.3":             false,
		"1.2.3.4":           true,
	}

	for s, b := range data {
		r := CheckIPv4Address(s)
		if r != b {
			t.Fatalf("错误: 数据 -- %s, 期待结果 %t, 计算结果 %t", s, b, r)
		}
	}
}

func TestCheckUserID(t *testing.T) {
	data := map[string]bool{
		"123456737AE8@etv1": false,
		"80560422@etv1":     true,
		"12345678@etv1":     true,
		"12345678@etv11":    false,
	}

	for s, b := range data {
		r := CheckUserID(s)
		if r != b {
			t.Fatalf("错误: 数据 -- %s, 期待结果 %t, 计算结果 %t", s, b, r)
		}
	}
}

func TestCheckSNCode(t *testing.T) {
	data := map[string]bool{
		"000100161231015063002D47":  true,
		"000200325207021201504CB6":  true,
		"000300325407022101504DC":   false,
		"000300325407022101504DC11": false,
		"001300325407022101504DC9":  false,
	}

	for s, b := range data {
		r := CheckSNCode(s)
		if r != b {
			t.Fatalf("错误: 数据 -- %s, 期待结果 %t, 计算结果 %t", s, b, r)
		}
	}
}
