package utils

import "testing"

func TestGenerateIP(t *testing.T) {
	t.Run("Right case", func(t *testing.T) {
		if GenerateIP("22.1.22.254", ",") != "022,001,022,254" {
			t.Fatalf("Err")
		}
	})

	t.Run("Wrong case", func(t *testing.T) {
		if GenerateIP("", ",") != "" {
			t.Fatalf("Err")
		}
	})
}

func TestGenerateRandomInt(t *testing.T) {
	for i := 0; i < 8; i++ {
		str := GenerateRandomInt(8)
		if str == "" {
			t.Fatalf("Err: Null string")
		}
		if len(str) != 8 {
			t.Fatalf("Err: Length not match")
		}
	}
}

func TestInsertStrInUserToken(t *testing.T) {
	before := "12345678@etv1***14670b274558c229"
	after := "123456737AE8@etv1***14670b274558c229"

	result := InsertStrInUserToken(before)

	if result != after {
		t.Fatalf("Err result str: %s", result)
	}
}

func TestRemoveStrInUserToken(t *testing.T) {
	after := "12345678@etv1***14670b274558c229"
	before := "123456737AE8@etv1***14670b274558c229"

	result := RemoveStrInUserToken(before)

	if result != after {
		t.Fatalf("Err result str: %s", result)
	}
}
