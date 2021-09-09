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

func TestAddWebhookRequestDecoder(t *testing.T) {

	type testCase struct {
		Description                 string
		InputPayload                string
		ExpectedLegacyDecodingCount float64
		ExpectedErr                 error
		ExpectedDecodedRequest      *addWebhookRequest
		ReadBodyFail                bool
		Validator                   Validator
	}

	tcs := []testCase{
		{
			Description:            "Normal happy path",
			InputPayload:           addWebhookDecoderInput(),
			ExpectedDecodedRequest: addWebhookDecoderOutput(),
			Validator:              Validators{},
		},
		{
			Description:                 "Legacy decoding",
			InputPayload:                addWebhookDecoderLegacyInput(),
			ExpectedLegacyDecodingCount: 1,
			ExpectedDecodedRequest:      addWebhookDecoderOutput(),
			Validator:                   Validators{},
		},
		{
			Description:  "Failed to JSON Unmarshal",
			InputPayload: "{",
			ExpectedErr:  errFailedWebhookUnmarshal,
			Validator:    Validators{},
		},
		{
			Description:  "Empty legacy case",
			InputPayload: "[]",
			ExpectedErr:  errNoWebhooksInLegacyDecode,
			Validator:    Validators{},
		},
		{
			Description:  "Webhook validation Failure",
			InputPayload: addWebhookDecoderInput(),
			Validator:    Validators{mockValidator()},
			ExpectedErr:  errMockValidatorFail,
		},
			Description:  "Request Body Read Failure",
			ExpectedErr:  errReadBodyFail,
			ReadBodyFail: true,
			Validator:    Validators{},
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
				v: tc.Validator,
			}
			decode := addWebhookRequestDecoder(config)

			r, err := http.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBufferString(tc.InputPayload))
			require.Nil(err)

			if tc.ReadBodyFail {
				r.Body = errReader{}
			}

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
				assert.True(errors.Is(err, tc.ExpectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.ExpectedErr))
				var s kithttp.StatusCoder
				isCoder := errors.As(err, &s)
				require.True(isCoder, "error isn't StatusCoder as expected")
				require.Equal(http.StatusBadRequest, s.StatusCode())
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
