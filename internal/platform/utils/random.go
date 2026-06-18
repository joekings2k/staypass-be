package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
)

func RandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return strings.TrimRight(hex.EncodeToString(b), "=")
}

func RandomPhoneNumber(n int) (string, error) {
	prefixes := []string{"70", "80", "81", "90", "91"}

	prefixIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(prefixes))))
	if err != nil {
		return "", err
	}

	prefix := prefixes[prefixIndex.Int64()]

	b := make([]byte, n)
	numbers := "0123456789"
	for i := range n {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(numbers))))
		if err != nil {
			return "", err
		}
		b[i] = numbers[randomIndex.Int64()]
	}
	result := "+234" + prefix + string(b)

	return result, nil
}

func RandomEmail() (string, error) {
	prefixm := RandomString(10)
	domainm := RandomString(5)

	return prefixm + "@" + domainm + ".com", nil
}

type RandomEstate struct {
	Name string
	Slug string
}

func RandomEstateAndSlug() (RandomEstate, error) {
	prefix := RandomString(10)
	suffix := "estate"

	return RandomEstate{
		Name: fmt.Sprintf("%s %s", prefix, suffix),
		Slug: strings.ToLower(
			fmt.Sprintf("%s-%d-estate", prefix, time.Now().UnixNano()),
		),
	}, nil
}
