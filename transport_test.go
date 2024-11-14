// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webhook-schema"
)

func TestErrorEncoder(t *testing.T) {
	mockHandlerConfig := HandlerConfig{}

	type testCase struct {
		Description  string
		InputErr     error
		ExpectedCode int
		HConfig      HandlerConfig
	}
	tcs := []testCase{
		{
			Description:  "Internal",
			InputErr:     errors.New("some failure"),
			HConfig:      mockHandlerConfig,
			ExpectedCode: 500,
		},
		{
			Description:  "Coded request",
			InputErr:     BadRequestErr{Message: "invalid param"},
			HConfig:      mockHandlerConfig,
			ExpectedCode: 400,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			recorder := httptest.NewRecorder()
			e := errorEncoder()
			e(context.Background(), tc.InputErr, recorder)
			assert.Equal(tc.ExpectedCode, recorder.Code)
			assert.JSONEq(fmt.Sprintf(`{"message": "%s"}`, tc.InputErr.Error()), recorder.Body.String())
			assert.Equal("application/json", recorder.Header().Get("Content-Type"))
		})
	}
}

func TestEncodeWebhookResponse(t *testing.T) {
	assert := assert.New(t)
	recorder := httptest.NewRecorder()
	encodeAddWebhookResponse(context.Background(), recorder, nil)
	assert.JSONEq(`{"message": "Success"}`, recorder.Body.String())
	assert.Equal(200, recorder.Code)
}

func TestGetOwner(t *testing.T) {
	type testCase struct {
		Description   string
		Auth          string
		ExpectedOwner string
	}

	tcs := []testCase{
		{
			Description:   "jwt token",
			Auth:          "jwt",
			ExpectedOwner: "test-subject",
		},
		{
			Description:   "basic token",
			Auth:          "basic",
			ExpectedOwner: "test-subject",
		},
	}
	for _, tc := range tcs {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert := assert.New(t)
		if tc.Auth != "" {
			AddAuth(tc.Auth, req, true, true)
		}
		// nolint:typecheck
		owner := getOwner(req)
		assert.Equal(tc.ExpectedOwner, owner)
	}
}

func TestEncodeGetAllWebhooksResponse(t *testing.T) {
	type testCase struct {
		Description           string
		InputInternalWebhooks []Register
		ExpectedJSONResp      string
		ExpectedErr           error
	}
	tcs := []testCase{
		{
			Description:           "Two webhooks",
			InputInternalWebhooks: encodeGetAllInput(),
			ExpectedJSONResp:      encodeGetAllOutput(),
		},
		{
			Description:      "Nil",
			ExpectedJSONResp: "[]",
		},
		{
			Description:      "Empty",
			ExpectedJSONResp: "[]",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			recorder := httptest.NewRecorder()
			err := encodeGetAllWebhooksResponse(context.Background(), recorder, tc.InputInternalWebhooks)
			assert.Nil(err)
			assert.Equal("application/json", recorder.Header().Get("Content-Type"))
			assert.JSONEq(tc.ExpectedJSONResp, recorder.Body.String())
		})
	}
}

func TestAddWebhookRequestDecoder(t *testing.T) {
	type testCase struct {
		Description            string
		InputPayload           string
		ExpectedErr            error
		ExpectedDecodedRequest *addWebhookRequest
		ReadBodyFail           bool
		Validator              webhook.Validators
		ExpectedStatusCode     int
		Auth                   string
		WrongContext           bool
		DisablePartnerIDs      bool
	}

	tcs := []testCase{
		{
			Description:            "Normal happy path",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              webhook.Validators{},
			Auth:                   "jwt",
		},
		{
			Description:            "Normal happy path using Duration",
			InputPayload:           addWebhookDecoderDurationInput(),
			ExpectedDecodedRequest: addWebhookDecoderDurationOutput(true),
			Validator:              webhook.Validators{},
			Auth:                   "jwt",
		},
		{
			Description:            "No validator provided",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Auth:                   "jwt",
		},
		{
			Description:            "Do not check PartnerIDs",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              webhook.Validators{},
			Auth:                   "jwt",
			DisablePartnerIDs:      true,
		},
		{
			Description:            "Auth token not present failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              webhook.Validators{},
			ExpectedErr:            errAuthNotPresent,
			WrongContext:           true,
		},
		{
			Description:            "basic auth",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              webhook.Validators{},
			Auth:                   "basic",
		},
		{
			Description:        "Failed to JSON Unmarshal",
			InputPayload:       "{",
			ExpectedErr:        errFailedWebhookUnmarshal,
			Validator:          webhook.Validators{},
			ExpectedStatusCode: 400,
			Auth:               "jwt",
		},
		{
			Description:        "Failed to JSON Unmarshal Type Error",
			InputPayload:       addWebhookDecoderUnmarshalingErrorInput(false),
			ExpectedErr:        errFailedWebhookUnmarshal,
			Validator:          webhook.Validators{},
			ExpectedStatusCode: 400,
			Auth:               "jwt",
		},
		{
			Description:        "Failed to JSON Unmarshal Invalid Duration Error",
			InputPayload:       addWebhookDecoderUnmarshalingErrorInput(true),
			ExpectedErr:        errFailedWebhookUnmarshal,
			Validator:          webhook.Validators{},
			ExpectedStatusCode: 400,
			Auth:               "jwt",
		},
		{
			Description:  "Webhook validation Failure",
			InputPayload: addWebhookDecoderInput(),
			Validator:    mockValidator(),
			ExpectedErr:  errMockValidatorFail,
			Auth:         "jwt",
		},
		{
			Description:        "Request Body Read Failure",
			ExpectedErr:        errReadBodyFail,
			ReadBodyFail:       true,
			Validator:          webhook.Validators{},
			ExpectedStatusCode: 0,
			Auth:               "jwt",
		},
		{
			Description:            "Auth token is not present",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              webhook.Validators{},
			ExpectedErr:            errAuthNotPresent,
		},
		{
			Description:            "jwt auth token has no allowedPartners failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              webhook.Validators{},
			Auth:                   "jwtnopartners",
			ExpectedErr:            errPartnerIDsDoNotExist,
		},
		{
			Description:            "jwt partners do not cast failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              webhook.Validators{},
			Auth:                   "jwtpartnersdonotcast",
			ExpectedErr:            errGettingPartnerIDs,
		},
		{
			Description:            "auth is not jwt or basic failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              webhook.Validators{},
			Auth:                   "authnotbasicorjwt",
			ExpectedErr:            errAuthIsNotOfTypeBasicOrJWT,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			counter := new(mockCounter)
			config := transportConfig{
				now: func() time.Time {
					return getRefTime()
				},
				v:                 tc.Validator,
				disablePartnerIDs: tc.DisablePartnerIDs,
			}
			decode := addWebhookRequestDecoder(config)
			var err error
			r, err := http.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBufferString(tc.InputPayload))
			require.Nil(err)
			if tc.ReadBodyFail {
				r.Body = errReader{}
			}

			switch tc.Auth {
			case "basic":
				AddAuth("basic", r, false, false)
			case "jwt":
				AddAuth("jwt", r, true, true)
			case "jwtnopartners":
				AddAuth("jwt", r, false, false)
			case "jwtpartnersdonotcast":
				AddAuth("jwt", r, true, false)
			case "authnotbasicorjwt":
				AddAuth("notbasicofjwt", r, false, false)
			}

			if tc.Auth == "basic" {
				r.Header[DefaultBasicPartnerIDsHeader] = []string{"comcast"}
			}
			r.RemoteAddr = "original-requester.example.net:443"

			var decodedRequest interface{}
			if tc.WrongContext {
				decodedRequest, err = decode(context.Background(), r)
			} else {
				decodedRequest, err = decode(r.Context(), r)
			}

			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.ExpectedErr))
				if tc.ExpectedStatusCode != 0 {
					var s kithttp.StatusCoder
					isCoder := errors.As(err, &s)
					require.True(isCoder, "error isn't StatusCoder as expected")
					require.Equal(tc.ExpectedStatusCode, s.StatusCode())
				}

			} else {
				assert.NoError(err)
				assert.EqualValues(tc.ExpectedDecodedRequest, decodedRequest)
			}

			// nolint:typecheck
			counter.AssertExpectations(t)
		})
	}
}

func addWebhookDecoderInput() string {
	return `
		{
			"registered_from_address": "original-requester.example.net:443",
			"config": {
				"url": "http://deliver-here-0.example.net",
				"content_type": "application/json",
				"secret": "superSecretXYZ"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "http://contact-here-when-fails.example.net",
			"duration": "0s",
			"until": "2021-01-02T15:04:10Z"
		}
	`
}
func addWebhookDecoderDurationInput() string {
	return `
		{
			"registered_from_address": "original-requester.example.net:443",
			"config": {
				"url": "http://deliver-here-0.example.net",
				"content_type": "application/json",
				"secret": "superSecretXYZ"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "http://contact-here-when-fails.example.net",
			"duration": "300s"
		}
	`
}

func addWebhookDecoderUnmarshalingErrorInput(duration bool) string {
	if duration {
		return `
		{
			"config": {
				"url": "http://deliver-here-0.example.net",
				"content_type": "application/json",
				"secret": "superSecretXYZ"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "http://contact-here-when-fails.example.net",
			"duration": "hehe",
			"until": "2021-01-02T15:04:10Z"
		}
	`
	}
	return `
		{
			"config": {
				"url": "http://deliver-here-0.example.net",
				"content_type": 5,
				"secret": "superSecretXYZ"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "http://contact-here-when-fails.example.net",
			"duration": 0,
			"until": "2021-01-02T15:04:10Z"
		}
	`
}

func addWebhookDecoderOutput(withPIDs bool) *addWebhookRequest {
	if withPIDs {
		return &addWebhookRequest{
			owner: "test-subject",
			internalWebook: &RegistryV1{
				Registration: webhook.RegistrationV1{
					Address: "original-requester.example.net:443",
					Config: webhook.DeliveryConfig{
						ReceiverURL: "http://deliver-here-0.example.net",
						ContentType: "application/json",
						Secret:      "superSecretXYZ",
					},
					Events: []string{"online"},
					Matcher: webhook.MetadataMatcherConfig{
						DeviceID: []string{"mac:aabbccddee.*"},
					},
					FailureURL: "http://contact-here-when-fails.example.net",
					Duration:   webhook.CustomDuration(0 * time.Second),
					Until:      getRefTime().Add(10 * time.Second),
				},
				PartnerIDs: []string{"comcast"},
			},
		}
	}
	return &addWebhookRequest{
		owner: "test-subject",
		internalWebook: &RegistryV1{
			Registration: webhook.RegistrationV1{
				Address: "original-requester.example.net:443",
				Config: webhook.DeliveryConfig{
					ReceiverURL: "http://deliver-here-0.example.net",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "http://contact-here-when-fails.example.net",
				Duration:   webhook.CustomDuration(0 * time.Second),
				Until:      getRefTime().Add(10 * time.Second),
			},
			PartnerIDs: []string{},
		},
	}
}
func addWebhookDecoderDurationOutput(withPIDs bool) *addWebhookRequest {
	if withPIDs {
		return &addWebhookRequest{
			owner: "test-subject",
			internalWebook: &RegistryV1{
				Registration: webhook.RegistrationV1{
					Address: "original-requester.example.net:443",
					Config: webhook.DeliveryConfig{
						ReceiverURL: "http://deliver-here-0.example.net",
						ContentType: "application/json",
						Secret:      "superSecretXYZ",
					},
					Events: []string{"online"},
					Matcher: webhook.MetadataMatcherConfig{
						DeviceID: []string{"mac:aabbccddee.*"},
					},
					FailureURL: "http://contact-here-when-fails.example.net",
					Duration:   webhook.CustomDuration(5 * time.Minute),
					Until:      getRefTime().Add(5 * time.Minute),
				},
				PartnerIDs: []string{"comcast"},
			},
		}
	}
	return &addWebhookRequest{
		owner: "test-subject",
		internalWebook: &RegistryV1{
			Registration: webhook.RegistrationV1{
				Address: "original-requester.example.net:443",
				Config: webhook.DeliveryConfig{
					ReceiverURL: "http://deliver-here-0.example.net",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "http://contact-here-when-fails.example.net",
				Duration:   webhook.CustomDuration(5 * time.Minute),
				Until:      getRefTime().Add(5 * time.Minute),
			},
			PartnerIDs: []string{},
		},
	}
}

func encodeGetAllInput() []Register {
	return []Register{
		&RegistryV1{
			Registration: webhook.RegistrationV1{
				Address: "http://original-requester.example.net",
				Config: webhook.DeliveryConfig{
					ReceiverURL: "http://deliver-here-0.example.net",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: struct {
					DeviceID []string `json:"device_id"`
				}{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "http://contact-here-when-fails.example.net",
				Duration:   webhook.CustomDuration(0 * time.Second),
				Until:      getRefTime().Add(10 * time.Second),
			},
			PartnerIDs: []string{"comcast"},
		},
		&RegistryV1{
			Registration: webhook.RegistrationV1{
				Address: "http://original-requester.example.net",
				Config: webhook.DeliveryConfig{
					ContentType: "application/json",
					ReceiverURL: "http://deliver-here-1.example.net",
					Secret:      "doNotShare:e=mc^2",
				},
				Events: []string{"online"},
				Matcher: struct {
					DeviceID []string `json:"device_id"`
				}{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "http://contact-here-when-fails.example.net",
				Duration:   webhook.CustomDuration(0 * time.Second),
				Until:      getRefTime().Add(20 * time.Second),
			},
			PartnerIDs: []string{"comcast"},
		},
	}

}

// once we move to go1.16 we could just embed this from a JSON file
// https://golang.org/doc/go1.16#library-embed
func encodeGetAllOutput() string {
	return `
	[
		{
			"registered_from_address": "http://original-requester.example.net",
			"config": {
				"url": "http://deliver-here-0.example.net",
				"content_type": "application/json",
				"secret": "<obfuscated>"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "http://contact-here-when-fails.example.net",
			"duration": "0s",
			"until": "2021-01-02T15:04:10Z"
		},
		{
			"registered_from_address": "http://original-requester.example.net",
			"config": {
				"url": "http://deliver-here-1.example.net",
				"content_type": "application/json",
				"secret": "<obfuscated>"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "http://contact-here-when-fails.example.net",
			"duration": "0s",
			"until": "2021-01-02T15:04:20Z"
		}
	]
	`
}

func TestSetWebhookDefaults(t *testing.T) {
	tcs := []struct {
		desc                 string
		registration         *webhook.RegistrationV1
		remoteAddr           string
		expectedRegistration *webhook.RegistrationV1
	}{
		{
			desc: "No Until, Address, or DeviceID",
			registration: &webhook.RegistrationV1{
				Config: webhook.DeliveryConfig{
					ReceiverURL: "https://deliver-here.example.net",
				},
				Events:   []string{"online", "offline"},
				Matcher:  webhook.MetadataMatcherConfig{},
				Duration: webhook.CustomDuration(5 * time.Minute),
			},
			remoteAddr: "http://original-requester.example.net",
			expectedRegistration: &webhook.RegistrationV1{
				Address: "http://original-requester.example.net",
				Config: webhook.DeliveryConfig{
					ReceiverURL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{".*"}},
				Duration: webhook.CustomDuration(5 * time.Minute),
				Until:    mockNow().Add(5 * time.Minute),
			},
		},
		{
			desc: "No Address or Request Address",
			registration: &webhook.RegistrationV1{
				Config: webhook.DeliveryConfig{
					ReceiverURL: "https://deliver-here.example.net",
				},
				Events:   []string{"online", "offline"},
				Matcher:  webhook.MetadataMatcherConfig{},
				Duration: webhook.CustomDuration(5 * time.Minute),
			},
			expectedRegistration: &webhook.RegistrationV1{
				Config: webhook.DeliveryConfig{
					ReceiverURL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{".*"}},
				Duration: webhook.CustomDuration(5 * time.Minute),
				Until:    mockNow().Add(5 * time.Minute),
			},
		},
		{
			desc: "All values set",
			registration: &webhook.RegistrationV1{
				Address: "requester.example.net:443",
				Config: webhook.DeliveryConfig{
					ReceiverURL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{".*"},
				},
				Duration: webhook.CustomDuration(5 * time.Minute),
				Until:    mockNow().Add(5 * time.Minute),
			},
			remoteAddr: "requester.example.net:443",
			expectedRegistration: &webhook.RegistrationV1{
				Address: "requester.example.net:443",
				Config: webhook.DeliveryConfig{
					ReceiverURL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{".*"},
				},
				Duration: webhook.CustomDuration(5 * time.Minute),
				Until:    mockNow().Add(5 * time.Minute),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			w := webhookValidator{
				now: mockNow,
			}
			w.setV1Defaults(tc.registration, tc.remoteAddr)
			assert.Equal(tc.expectedRegistration, tc.registration)
		})
	}
}

type BadRequestErr struct {
	Message string
}

func (bre BadRequestErr) Error() string {
	return bre.Message
}

func (bre BadRequestErr) SanitizedError() string {
	return bre.Message
}

func (bre BadRequestErr) StatusCode() int {
	return http.StatusBadRequest
}

func createJWT(hasResources, hasPartners bool) ([]byte, error) {
	allowedResources := make(map[string]any)
	if hasResources && hasPartners {
		allowedResources["allowedPartners"] = []string{"comcast"}
	} else if hasResources {
		allowedResources["allowedPartners"] = []string{}
	}
	audience := []string{"test-audience"}
	capabilities := []string{
		"x1:webpa:api:.*:all",
		"x1:webpa:api:device/.*/config\\b:all",
	}
	issuedAt := time.Now().Add(-time.Second).Round(time.Second).UTC()

	testToken, err := jwt.NewBuilder().
		Audience(audience).
		Subject("test-subject").
		IssuedAt(issuedAt).
		Expiration(issuedAt.Add(time.Hour)).
		NotBefore(issuedAt.Add(-time.Hour)).
		JwtID("test-jwt").
		Issuer("test-issuer").
		Claim("capabilities", capabilities).
		Claim("allowedResources", allowedResources).
		Claim("version", "2.0").
		Build()

	if err != nil {
		return nil, err
	}
	signed, err := jwt.Sign(testToken, jwt.WithKey(jwa.RS256, initializeKey()))

	return signed, err
}

func AddAuth(auth string, req *http.Request, hasResources, hasPartners bool) error {
	if auth == "jwt" {
		signed, err := createJWT(hasResources, hasPartners)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", "Bearer "+string(signed))
	} else if auth == "basic" {
		req.SetBasicAuth("test-subject", "test-password")
	}
	return errAuthIsNotOfTypeBasicOrJWT
}

func initializeKey() jwk.Key {
	key, _ := jwk.ParseKey([]byte(`{
    "p": "7HMYtb-1dKyDp1OkdKc9WDdVMw3vtiiKDyuyRwnnwMOoYLPYxqE0CUMzw8_zXuzq7WJAmGiFd5q7oVzkbHzrtQ",
    "kty": "RSA",
    "q": "5253lCAgBLr8SR_VzzDtk_3XTHVmVIgniajMl7XM-ttrUONV86DoIm9VBx6ywEKpj5Xv3USBRNlpf8OXqWVhPw",
    "d": "G7RLbBiCkiZuepbu46G0P8J7vn5l8G6U78gcMRdEhEsaXGZz_ZnbqjW6u8KI_3akrBT__GDPf8Hx8HBNKX5T9jNQW0WtJg1XnwHOK_OJefZl2fnx-85h3tfPD4zI3m54fydce_2kDVvqTOx_XXdNJD7v5TIAgvCymQv7qvzQ0VE",
    "e": "AQAB",
    "use": "sig",
    "kid": "test",
    "qi": "a_6YlMdA9b6piRodA0MR7DwjbALlMan19wj_VkgZ8Xoilq68sGaV2CQDoAdsTW9Mjt5PpCxvJawz0AMr6LIk9w",
    "dp": "s55HgiGs_YHjzSOsBXXaEv6NuWf31l_7aMTf_DkZFYVMjpFwtotVFUg4taJuFYlSeZwux9h2s0IXEOCZIZTQFQ",
    "alg": "RS256",
    "dq": "M79xoX9laWleDAPATSnFlbfGsmP106T2IkPKK4oNIXJ6loWerHEoNrrqKkNk-LRvMZn3HmS4-uoaOuVDPi9bBQ",
    "n": "1cHjMu7H10hKxnoq3-PJT9R25bkgVX1b39faqfecC82RMcD2DkgCiKGxkCmdUzuebpmXCZuxp-rVVbjrnrI5phAdjshZlkHwV0tyJOcerXsPgu4uk_VIJgtLdvgUAtVEd8-ZF4Y9YNOAKtf2AHAoRdP0ZVH7iVWbE6qU-IN2los"
}`))
	return key
}
