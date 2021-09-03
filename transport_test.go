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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/httpaux/erraux"
)

func TestErrorEncoder(t *testing.T) {
	type testCase struct {
		Description  string
		InputErr     error
		ExpectedCode int
	}
	tcs := []testCase{
		{
			Description:  "Internal",
			InputErr:     errors.New("some failure"),
			ExpectedCode: 500,
		},
		{
			Description:  "Coded request",
			InputErr:     store.BadRequestErr{Message: "invalid param"},
			ExpectedCode: 400,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			recorder := httptest.NewRecorder()
			errorEncoder(context.Background(), tc.InputErr, recorder)

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
		Description      string
		InputWebhooks    []Webhook
		ExpectedJSONResp string
		ExpectedErr      error
	}
	tcs := []testCase{
		{
			Description:      "Two webhooks",
			InputWebhooks:    encodeGetAllInput(),
			ExpectedJSONResp: encodeGetAllOutput(),
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
			err := encodeGetAllWebhooksResponse(context.Background(), recorder, tc.InputWebhooks)
			assert.Nil(err)
			assert.Equal("application/json", recorder.Header().Get("Content-Type"))
			assert.JSONEq(tc.ExpectedJSONResp, recorder.Body.String())
		})
	}
}

func TestValidateWebhook(t *testing.T) {
	type testCase struct {
		Description     string
		InputWebhook    *Webhook
		ExpectedErr     *erraux.Error
		ExpectedWebhook *Webhook
	}

	nowSnapShot := time.Now()
	tcs := []testCase{
		{
			Description: "No config url",
			InputWebhook: &Webhook{
				Config: DeliveryConfig{
					ContentType: "application/json",
				},
			},
			ExpectedErr: &erraux.Error{Err: errInvalidConfigURL, Code: 400},
		},
		{
			Description: "No events",
			InputWebhook: &Webhook{
				Config: DeliveryConfig{
					URL:         "https://deliver-here.example.net",
					ContentType: "application/json",
				},
			},
			ExpectedErr: &erraux.Error{Err: errInvalidEvents, Code: 400},
		},
		{
			Description: "Valid defaulted values",
			InputWebhook: &Webhook{
				Config: DeliveryConfig{
					URL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
			},
			ExpectedWebhook: &Webhook{
				Address: "requester.example.net",
				Config: DeliveryConfig{
					URL: "https://deliver-here.example.net",
				},
				Events: []string{"online", "offline"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{".*"},
				},
				Duration: 5 * time.Minute,
				Until:    nowSnapShot.Add(5 * time.Minute),
			},
		},
		{
			Description:     "Provided values",
			InputWebhook:    validateWebhookInput(nowSnapShot),
			ExpectedWebhook: validateWebhookOutput(nowSnapShot),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			wv := webhookValidator{
				now: func() time.Time {
					return nowSnapShot
				},
			}
			err := wv.setWebhookDefaults(tc.InputWebhook, "requester.example.net:443")

			if tc.ExpectedErr != nil {
				assert.EqualValues(tc.ExpectedErr, err)
			} else {
				assert.Nil(err)
				assert.EqualValues(tc.ExpectedWebhook, tc.InputWebhook)
			}
		})
	}

}

func TestAddWebhookRequestDecoder(t *testing.T) {
	type testCase struct {
		Description                 string
		InputPayload                string
		ExpectedLegacyDecodingCount float64
		ExpectedErr                 error
		ExpectedDecodedRequest      *addWebhookRequest
	}

	tcs := []testCase{
		{
			Description:            "Normal happy path",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(),
		},
		{
			Description:                 "Legacy decoding",
			InputPayload:                addWebhookDecoderLegacyInput(),
			ExpectedLegacyDecodingCount: 1,
			ExpectedDecodedRequest:      addWebhookDecoderOutput(),
		},
		{
			Description:  "Failed to JSON Unmarshal",
			InputPayload: "{",
			ExpectedErr:  &erraux.Error{Err: errFailedWebhookUnmarshal, Code: http.StatusBadRequest},
		},
		{
			Description:  "Empty legacy case",
			InputPayload: "[]",
			ExpectedErr:  &erraux.Error{Err: errNoWebhooksInLegacyDecode, Code: http.StatusBadRequest},
		},
		{
			Description:  "Invalid Input",
			InputPayload: `{"events": ["online", "offline"]}`,
			ExpectedErr:  &erraux.Error{Code: http.StatusBadRequest, Err: errInvalidConfigURL},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			counter := new(mockCounter)
			config := transportConfig{
				webhookLegacyDecodeCount: counter,
				now: func() time.Time {
					return getRefTime()
				},
			}
			decode := addWebhookRequestDecoder(config)
			r, err := http.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBufferString(tc.InputPayload))
			require.Nil(err)

			auth := bascule.Authentication{
				Token: bascule.NewToken("jwt", "owner-from-auth", nil),
			}
			ctx := bascule.WithAuthentication(r.Context(), auth)
			r = r.WithContext(ctx)
			r.RemoteAddr = "original-requester.example.net:443"

			if tc.ExpectedLegacyDecodingCount > 0 {
				counter.On("With", URLLabel, tc.ExpectedDecodedRequest.webhook.Config.URL).Times(int(tc.ExpectedLegacyDecodingCount))
				counter.On("Add", float64(1)).Times(int(tc.ExpectedLegacyDecodingCount))
			}

			decodedRequest, err := decode(context.Background(), r)
			if tc.ExpectedErr != nil {
				assert.Equal(tc.ExpectedErr, err)
			} else {
				assert.Nil(err)
				assert.EqualValues(tc.ExpectedDecodedRequest, decodedRequest)
			}

			if tc.ExpectedLegacyDecodingCount < 1 {
				counter.AssertNotCalled(t, "With")
				counter.AssertNotCalled(t, "Add")
			}

			counter.AssertExpectations(t)
		})
	}
}

func validateWebhookInput(nowSnapShot time.Time) *Webhook {
	return &Webhook{
		Address: "requester.example.net",
		Config: DeliveryConfig{
			URL: "https://deliver-here.example.net",
		},
		Events: []string{"online", "offline"},
		Matcher: MetadataMatcherConfig{
			DeviceID: []string{".*"},
		},
		Duration: 25 * time.Minute,
		Until:    nowSnapShot.Add(1000 * time.Hour),
	}
}

func validateWebhookOutput(nowSnapShot time.Time) *Webhook {
	webhook := validateWebhookInput(nowSnapShot)
	webhook.Duration = defaultWebhookExpiration
	return webhook
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

func addWebhookDecoderLegacyInput() string {
	return `
	[
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
		},
		{
			"config": {
				"url": "http://deliver-here-1.example.net",
				"content_type": "application/json",
				"secret": "superSecretXYZ"
			},
			"events": ["online"]
		}
	]
	`
}

func addWebhookDecoderOutput() *addWebhookRequest {
	return &addWebhookRequest{
		owner: "owner-from-auth",
		webhook: Webhook{
			Address: "original-requester.example.net",
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
			Until:      getRefTime().Add(10 * time.Second),
		},
	}

}

func encodeGetAllInput() []Webhook {
	return []Webhook{
		{
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
		{
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
