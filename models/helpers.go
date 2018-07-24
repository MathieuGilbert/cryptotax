package models

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Random returns a random, url safe value of the bit length passed in
// https://tech.townsourced.com/post/anatomy-of-a-go-web-app-authentication/
func Random(bits int) string {
	result := make([]byte, bits/8)
	_, err := io.ReadFull(rand.Reader, result)
	if err != nil {
		panic(fmt.Sprintf("Error generating random values: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(result)
}
