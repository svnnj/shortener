package service

import (
	"crypto/rand"
	"encoding/base64"
)

type (
	TokenService interface {
		Generate() string
	}

	b64TokenGen struct {
		randLen int
	}
)

func NewB64TokenGen(randLen int) *b64TokenGen {
	return &b64TokenGen{
		randLen: randLen,
	}
}

func (t *b64TokenGen) Generate() string {
	b := make([]byte, t.randLen)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
