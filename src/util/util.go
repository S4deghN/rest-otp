package util

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"
	"cmp"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateOTP() string {
	n, err := rand.Int(rand.Reader, big.NewInt(999999))
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%06d", n)
}

func SendOTP(phoneNumber, otp string) error {
	log.Printf("Phone Number: %s, OTP: %s\n", phoneNumber, otp)
	return nil
}

func GenerateJWT(userId string) (string, error) {
	key := []byte("very-secret-not-plain-text-stored-key")
	claims := jwt.MapClaims{
		"sub": userId,
		"iat": jwt.NewNumericDate(time.Now()),
		"exp": jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(key)
}

func Clamp[T cmp.Ordered](x, min, max T) T {
	if cmp.Compare(x, min) < 0 {
		return min
	}
	if cmp.Compare(x, max) > 0 {
		return max
	}
	return x
}
