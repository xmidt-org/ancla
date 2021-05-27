package ancla

import (
	"errors"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/spf13/cast"
	"github.com/xmidt-org/bascule/acquire"
)

type jwtAcquireParserType string

const (
	simpleType jwtAcquireParserType = "simple"
	rawType    jwtAcquireParserType = "raw"
)

var (
	errMissingExpClaim   = errors.New("missing exp claim in jwt")
	errUnexpectedCasting = errors.New("unexpected casting error")
)

type jwtAcquireParser struct {
	token      acquire.TokenParser
	expiration acquire.ParseExpiration
}

func rawTokenParser(data []byte) (string, error) {
	return string(data), nil
}

func rawTokenExpirationParser(data []byte) (time.Time, error) {
	p := jwt.Parser{SkipClaimsValidation: true}
	token, _, err := p.ParseUnverified(string(data), jwt.MapClaims{})
	if err != nil {
		return time.Time{}, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return time.Time{}, errUnexpectedCasting
	}
	expVal, ok := claims["exp"]
	if !ok {
		return time.Time{}, errMissingExpClaim
	}

	exp, err := cast.ToInt64E(expVal)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(exp, 0), nil
}

func newJWTAcquireParser(pType jwtAcquireParserType) (jwtAcquireParser, error) {
	if pType == "" {
		pType = simpleType
	}
	if pType != simpleType && pType != rawType {
		return jwtAcquireParser{}, errors.New("only 'simple' or 'raw' are supported as jwt acquire parser types")
	}
	parser := jwtAcquireParser{}
	if pType == simpleType {
		parser.token = acquire.DefaultTokenParser
		parser.expiration = acquire.DefaultExpirationParser
		return parser, nil
	}
	parser.token = rawTokenParser
	parser.expiration = rawTokenExpirationParser
	return parser, nil
}
