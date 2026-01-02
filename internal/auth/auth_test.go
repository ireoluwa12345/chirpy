package auth

import (
	"fmt"
	"regexp"
	"testing"
)

// TestHashPassword checks that the hashPassword function produces a valid Argon2id hash.
func TestHashPassword(t *testing.T) {
	password := "password"
	want := regexp.MustCompile(`^\$argon2id\$v=\d+\$m=\d+,t=\d+,p=\d+\$[A-Za-z0-9+/]+={0,2}\$[A-Za-z0-9+/]+={0,2}$`)
	hash, err := hashPassword(password)
	if err != nil {
		t.Errorf(`hashPassword("password") = %q, %v, want match for %#q, nil`, hash, err, want)
	}
}

// TestVerifyPassword checks that the verifyPassword function correctly verifies a password against its hash.
func TestVerifyPassword(t *testing.T) {
	password := "password"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword(%q) unexpected error: %v", password, err)
	}

	ok, err := verifyPassword("hello", hash)
	fmt.Println(err)
	if err != nil {
		t.Errorf(`verifyPassword("password", %q) = %v, %v, want true, nil`, hash, ok, err)
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	password := "password"
	wrongPassword := "wrongpassword"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword(%q) unexpected error: %v", password, err)
	}

	ok, err := verifyPassword(wrongPassword, hash)
	if err != nil {
		t.Errorf(`verifyPassword("wrongpassword", %q) = %v, %v, want false, nil`, hash, ok, err)
	}
	if ok {
		t.Errorf(`verifyPassword("wrongpassword", %q) = true, nil, want false, nil`, hash)
	}
}
