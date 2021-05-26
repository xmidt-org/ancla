package ancla

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewJWTAcquireParser(t *testing.T) {
	tcs := []struct {
		Description string
		ParserType  jwtAcquireParserType
		ShouldFail  bool
	}{
		{
			Description: "Default",
		},
		{
			Description: "Invalid type",
			ParserType:  "advanced",
			ShouldFail:  true,
		},
		{
			Description: "Simple",
			ParserType:  simpleType,
		},
		{
			Description: "Raw",
			ParserType:  rawType,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			p, err := newJWTAcquireParser(tc.ParserType)
			if tc.ShouldFail {
				assert.NotNil(err)
				assert.Nil(p.expiration)
				assert.Nil(p.token)
			} else {
				assert.Nil(err)
				assert.NotNil(p.expiration)
				assert.NotNil(p.token)
			}
		})
	}
}

//TODO: test raw token parser

//TODO: test raw token expiration parser
