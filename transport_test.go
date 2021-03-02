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
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
			InputWebhooks:    getTwoWebhooks(),
			ExpectedJSONResp: getTwoWebhooksJSONString(),
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

// once we move to go1.16 we could just embed this from a JSON file
// https://golang.org/doc/go1.16#library-embed
func getTwoWebhooksJSONString() string {
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

func getTwoWebhooks() []Webhook {
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
