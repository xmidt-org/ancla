/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/bascule"
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
			InputErr:     store.BadRequestErr{Message: "invalid param"},
			HConfig:      mockHandlerConfig,
			ExpectedCode: 400,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			recorder := httptest.NewRecorder()
			e := errorEncoder(tc.HConfig.GetLogger)
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
		Token         bascule.Token
		ExpectedOwner string
	}

	tcs := []testCase{
		{
			Description:   "No auth token",
			Token:         nil,
			ExpectedOwner: "",
		},
		{
			Description:   "jwt token",
			Token:         bascule.NewToken("jwt", "sub-value-001", nil),
			ExpectedOwner: "sub-value-001",
		},
		{
			Description:   "basic token",
			Token:         bascule.NewToken("basic", "user-001", nil),
			ExpectedOwner: "user-001",
		},

		{
			Description:   "unsupported",
			Token:         bascule.NewToken("badType", "principalVal", nil),
			ExpectedOwner: "",
		},
	}
	for _, tc := range tcs {
		assert := assert.New(t)
		var ctx = context.Background()
		if tc.Token != nil {
			auth := bascule.Authentication{
				Token: tc.Token,
			}
			ctx = bascule.WithAuthentication(ctx, auth)
		}
		owner := getOwner(ctx)
		assert.Equal(tc.ExpectedOwner, owner)
	}
}

func TestEncodeGetAllWebhooksResponse(t *testing.T) {
	type testCase struct {
		Description           string
		InputInternalWebhooks []InternalWebhook
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
		Validator              Validator
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
			Validator:              Validators{},
			Auth:                   "jwt",
		},
		{
			Description:            "Normal happy path using Duration",
			InputPayload:           addWebhookDecoderDurationInput(),
			ExpectedDecodedRequest: addWebhookDecoderDurationOutput(true),
			Validator:              Validators{},
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
			Validator:              Validators{},
			Auth:                   "jwt",
			DisablePartnerIDs:      true,
		},
		{
			Description:            "Auth token not present failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              Validators{},
			ExpectedErr:            errAuthNotPresent,
			WrongContext:           true,
		},
		{
			Description:            "Auth token is nil failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              Validators{},
			ExpectedErr:            errAuthTokenIsNil,
		},
		{
			Description:            "jwt auth token has no allowedPartners failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              Validators{},
			Auth:                   "jwtnopartners",
			ExpectedErr:            errPartnerIDsDoNotExist,
		},
		{
			Description:            "jwt partners do not cast failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              Validators{},
			Auth:                   "jwtpartnersdonotcast",
			ExpectedErr:            errGettingPartnerIDs,
		},
		{
			Description:            "auth is not jwt or basic failure",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              Validators{},
			Auth:                   "authnotbasicorjwt",
			ExpectedErr:            errAuthIsNotOfTypeBasicOrJWT,
		},
		{
			Description:            "basic auth",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(true),
			Validator:              Validators{},
			Auth:                   "basic",
		},
		{
			Description:        "Failed to JSON Unmarshal",
			InputPayload:       "{",
			ExpectedErr:        errFailedWebhookUnmarshal,
			Validator:          Validators{},
			ExpectedStatusCode: 400,
			Auth:               "jwt",
		},
		{
			Description:        "Failed to JSON Unmarshal Type Error",
			InputPayload:       addWebhookDecoderUnmarshalingErrorInput(false),
			ExpectedErr:        errFailedWebhookUnmarshal,
			Validator:          Validators{},
			ExpectedStatusCode: 400,
			Auth:               "jwt",
		},
		{
			Description:        "Failed to JSON Unmarshal Invalid Duration Error",
			InputPayload:       addWebhookDecoderUnmarshalingErrorInput(true),
			ExpectedErr:        errFailedWebhookUnmarshal,
			Validator:          Validators{},
			ExpectedStatusCode: 400,
			Auth:               "jwt",
		},
		{
			Description:  "Webhook validation Failure",
			InputPayload: addWebhookDecoderInput(),
			Validator:    Validators{mockValidator()},
			ExpectedErr:  errMockValidatorFail,
			Auth:         "jwt",
		},
		{
			Description:        "Request Body Read Failure",
			ExpectedErr:        errReadBodyFail,
			ReadBodyFail:       true,
			Validator:          Validators{},
			ExpectedStatusCode: 0,
			Auth:               "jwt",
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
			var auth bascule.Authentication

			switch tc.Auth {
			case "basic":
				auth = bascule.Authentication{
					Token: bascule.NewToken("basic", "owner-from-auth", bascule.NewAttributes(
						map[string]interface{}{})),
				}
			case "jwt":
				auth = bascule.Authentication{
					Token: bascule.NewToken("jwt", "owner-from-auth", bascule.NewAttributes(
						map[string]interface{}{"allowedResources": map[string]interface{}{"allowedPartners": "comcast"}})),
				}
			case "jwtnopartners":
				auth = bascule.Authentication{
					Token: bascule.NewToken("jwt", "owner-from-auth", bascule.NewAttributes(
						map[string]interface{}{})),
				}
			case "jwtpartnersdonotcast":
				auth = bascule.Authentication{
					Token: bascule.NewToken("jwt", "owner-from-auth", bascule.NewAttributes(
						map[string]interface{}{"allowedResources": map[string]interface{}{"allowedPartners": nil}})),
				}
			case "authnotbasicorjwt":
				auth = bascule.Authentication{
					Token: bascule.NewToken("spongebob", "owner-from-auth", bascule.NewAttributes(
						map[string]interface{}{})),
				}
			}

			r, err := http.NewRequestWithContext(bascule.WithAuthentication(context.Background(), auth),
				http.MethodPost, "http://localhost:8080", bytes.NewBufferString(tc.InputPayload))
			require.Nil(err)
			if tc.ReadBodyFail {
				r.Body = errReader{}
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

			counter.AssertExpectations(t)
		})
	}
}

func addWebhookDecoderInput() string {
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
			"duration": 0,
			"until": "2021-01-02T15:04:10Z"
		}
	`
}
func addWebhookDecoderDurationInput() string {
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
			"duration": 300
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
			owner: "owner-from-auth",
			internalWebook: InternalWebhook{
				Webhook: Webhook{
					Address: "original-requester.example.net:443",
					Config: DeliveryConfig{
						URL:         "http://deliver-here-0.example.net",
						ContentType: "application/json",
						Secret:      "superSecretXYZ",
					},
					Events: []string{"online"},
					Matcher: MetadataMatcherConfig{
						DeviceID: []string{"mac:aabbccddee.*"},
					},
					FailureURL: "http://contact-here-when-fails.example.net",
					Duration:   0,
					Until:      getRefTime().Add(10 * time.Second),
				},
				PartnerIDs: []string{"comcast"},
			},
		}
	}
	return &addWebhookRequest{
		owner: "owner-from-auth",
		internalWebook: InternalWebhook{
			Webhook: Webhook{
				Address: "original-requester.example.net:443",
				Config: DeliveryConfig{
					URL:         "http://deliver-here-0.example.net",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "http://contact-here-when-fails.example.net",
				Duration:   0,
				Until:      getRefTime().Add(10 * time.Second),
			},
			PartnerIDs: []string{},
		},
	}
}
func addWebhookDecoderDurationOutput(withPIDs bool) *addWebhookRequest {
	if withPIDs {
		return &addWebhookRequest{
			owner: "owner-from-auth",
			internalWebook: InternalWebhook{
				Webhook: Webhook{
					Address: "original-requester.example.net:443",
					Config: DeliveryConfig{
						URL:         "http://deliver-here-0.example.net",
						ContentType: "application/json",
						Secret:      "superSecretXYZ",
					},
					Events: []string{"online"},
					Matcher: MetadataMatcherConfig{
						DeviceID: []string{"mac:aabbccddee.*"},
					},
					FailureURL: "http://contact-here-when-fails.example.net",
					Duration:   5 * time.Minute,
					Until:      getRefTime().Add(5 * time.Minute),
				},
				PartnerIDs: []string{"comcast"},
			},
		}
	}
	return &addWebhookRequest{
		owner: "owner-from-auth",
		internalWebook: InternalWebhook{
			Webhook: Webhook{
				Address: "original-requester.example.net:443",
				Config: DeliveryConfig{
					URL:         "http://deliver-here-0.example.net",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "http://contact-here-when-fails.example.net",
				Duration:   5 * time.Minute,
				Until:      getRefTime().Add(5 * time.Minute),
			},
			PartnerIDs: []string{},
		},
	}
}

func encodeGetAllInput() []InternalWebhook {
	return []InternalWebhook{
		{
			Webhook: Webhook{
				Address: "http://original-requester.example.net",
				Config: DeliveryConfig{
					URL:         "http://deliver-here-0.example.net",
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
				Until:      getRefTime().Add(10 * time.Second),
			},
			PartnerIDs: []string{"comcast"},
		},
		{
			Webhook: Webhook{
				Address: "http://original-requester.example.net",
				Config: DeliveryConfig{
					ContentType: "application/json",
					URL:         "http://deliver-here-1.example.net",
					Secret:      "doNotShare:e=mc^2",
				},
				Events: []string{"online"},
				Matcher: struct {
					DeviceID []string `json:"device_id"`
				}{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "http://contact-here-when-fails.example.net",
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
			"duration": 0,
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
			"duration": 0,
			"until": "2021-01-02T15:04:20Z"
		}
	]
	`
}

func TestSetWebhookDefaults(t *testing.T) {
	tcs := []struct {
		desc            string
		webhook         Webhook
		remoteAddr      string
		expectedWebhook Webhook
	}{
		{
			desc: "No Until, Address, or DeviceID",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL: "https://deliver-here.example.net",
				},
				Events:   []string{"online", "offline"},
				Matcher:  MetadataMatcherConfig{},
				Duration: 5 * time.Minute,
			},
			remoteAddr: "http://original-requester.example.net",
			expectedWebhook: Webhook{
				Address: "http://original-requester.example.net",
				Config: DeliveryConfig{
					URL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{".*"}},
				Duration: 5 * time.Minute,
				Until:    mockNow().Add(5 * time.Minute),
			},
		},
		{
			desc: "No Address or Request Address",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL: "https://deliver-here.example.net",
				},
				Events:   []string{"online", "offline"},
				Matcher:  MetadataMatcherConfig{},
				Duration: 5 * time.Minute,
			},
			expectedWebhook: Webhook{
				Config: DeliveryConfig{
					URL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{".*"}},
				Duration: 5 * time.Minute,
				Until:    mockNow().Add(5 * time.Minute),
			},
		},
		{
			desc: "All values set",
			webhook: Webhook{
				Address: "requester.example.net:443",
				Config: DeliveryConfig{
					URL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{".*"},
				},
				Duration: 5 * time.Minute,
				Until:    mockNow().Add(5 * time.Minute),
			},
			remoteAddr: "requester.example.net:443",
			expectedWebhook: Webhook{
				Address: "requester.example.net:443",
				Config: DeliveryConfig{
					URL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{".*"},
				},
				Duration: 5 * time.Minute,
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
			w.setWebhookDefaults(&tc.webhook, tc.remoteAddr)
			assert.Equal(tc.expectedWebhook, tc.webhook)
		})
	}
}
