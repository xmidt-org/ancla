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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/ancla/auth"
	"github.com/xmidt-org/ancla/schema"
	webhook "github.com/xmidt-org/webhook-schema"
	"go.uber.org/zap"
)

var (
	mockNow = func() time.Time {
		return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	}
)

func TestErrorEncoder(t *testing.T) {
	mockHandlerConfig := HandlerConfig{GetLogger: func(context.Context) *zap.Logger {
		return zap.NewNop()
	}}

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
			// TODO: remove gokit from errorEncoder and then update TestErrorEncoder tests
			e := errorEncoder(tc.HConfig.GetLogger)
			e(context.Background(), tc.InputErr, recorder)
			assert.Equal(tc.ExpectedCode, recorder.Code)
			assert.JSONEq(fmt.Sprintf(`{"message": "%s"}`, tc.InputErr.Error()), recorder.Body.String())
			assert.Equal("application/json", recorder.Header().Get("Content-Type"))
		})
	}
}

func TestEncodeWRPEventStreamResponse(t *testing.T) {
	assert := assert.New(t)
	recorder := httptest.NewRecorder()
	encodeAddWRPEventStreamResponse(context.Background(), recorder, nil)
	assert.JSONEq(`{"message": "Success"}`, recorder.Body.String())
	assert.Equal(200, recorder.Code)
}

func TestEncodeGetAllWRPEventStreamsResponse(t *testing.T) {
	type testCase struct {
		Description      string
		InputSchemas     []schema.Manifest
		ExpectedJSONResp string
		ExpectedErr      error
	}
	tcs := []testCase{
		{
			Description:      "Two wrpEventStreams",
			InputSchemas:     encodeGetAllInput(),
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
			err := encodeGetAllWRPEventStreamsResponse(context.Background(), recorder, tc.InputSchemas)
			assert.Nil(err)
			assert.Equal("application/json", recorder.Header().Get("Content-Type"))
			assert.JSONEq(tc.ExpectedJSONResp, recorder.Body.String())
		})
	}
}

func TestAddWRPEventStreamRequestDecoder(t *testing.T) {
	type testCase struct {
		Description            string
		InputPayload           string
		ExpectedErr            error
		ExpectedDecodedRequest *addWRPEventStreamRequest
		ReadBodyFail           bool
		Validator              webhook.Validators
		ExpectedStatusCode     int
		Context                context.Context
		WrongContext           bool
		DisablePartnerIDs      bool
	}

	var (
		ctxEmpty                   = context.Background()
		ctxWithoutPartnerIDs       = auth.SetPrincipal(ctxEmpty, "owner-from-auth")
		ctxWithPrincipalPartnerIDs = auth.SetPartnerIDs(auth.SetPrincipal(ctxEmpty, "owner-from-auth"), []string{"comcast"})
	)

	tcs := []testCase{
		{
			Description:            "Normal happy path",
			InputPayload:           addWRPEventStreamDecoderInput(),
			ExpectedDecodedRequest: addWRPEventStreamDecoderOutput(true),
			Validator:              webhook.Validators{},
			Context:                ctxWithPrincipalPartnerIDs,
		},
		{
			Description:            "Normal happy path using Duration",
			InputPayload:           addWRPEventStreamDecoderDurationInput(),
			ExpectedDecodedRequest: addWRPEventStreamDecoderDurationOutput(true),
			Validator:              webhook.Validators{},
			Context:                ctxWithPrincipalPartnerIDs,
		},
		{
			Description:            "No validator provided",
			InputPayload:           addWRPEventStreamDecoderInput(),
			ExpectedDecodedRequest: addWRPEventStreamDecoderOutput(true),
			Context:                ctxWithPrincipalPartnerIDs,
		},
		{
			Description:            "Do not check PartnerIDs",
			InputPayload:           addWRPEventStreamDecoderInput(),
			ExpectedDecodedRequest: addWRPEventStreamDecoderOutput(false),
			Validator:              webhook.Validators{},
			Context:                ctxWithoutPartnerIDs,
			DisablePartnerIDs:      true,
		},
		{
			Description:            "unable to retrieve PartnerIDs failure",
			InputPayload:           addWRPEventStreamDecoderInput(),
			ExpectedDecodedRequest: addWRPEventStreamDecoderOutput(true),
			Validator:              webhook.Validators{},
			Context:                ctxWithoutPartnerIDs,
			ExpectedErr:            errGettingPartnerIDs,
		},
		{
			Description:        "Failed to JSON Unmarshal Type Error",
			InputPayload:       addWRPEventStreamDecoderUnmarshalingErrorInput(false),
			ExpectedErr:        errFailedWRPEventStreamUnmarshal,
			Validator:          webhook.Validators{},
			ExpectedStatusCode: 400,
			Context:            ctxWithPrincipalPartnerIDs,
		},
		{
			Description:        "Failed to JSON Unmarshal Invalid Duration Error",
			InputPayload:       addWRPEventStreamDecoderUnmarshalingErrorInput(true),
			ExpectedErr:        errFailedWRPEventStreamUnmarshal,
			Validator:          webhook.Validators{},
			ExpectedStatusCode: 400,
			Context:            ctxWithPrincipalPartnerIDs,
		},
		{
			Description:  "EventStream validation Failure",
			InputPayload: addWRPEventStreamDecoderInput(),
			Validator:    mockValidator(),
			Context:      ctxWithPrincipalPartnerIDs,
			ExpectedErr:  errMockValidatorFail,
		},
		{
			Description:        "Request Body Read Failure",
			ExpectedErr:        errReadBodyFail,
			ReadBodyFail:       true,
			Validator:          webhook.Validators{},
			Context:            ctxWithPrincipalPartnerIDs,
			ExpectedStatusCode: 0,
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
			decode := addWRPEventStreamRequestDecoder(config)
			var err error
			r, err := http.NewRequest(http.MethodPost, "http://localhost:8080", bytes.NewBufferString(tc.InputPayload))
			require.Nil(err)
			if tc.ReadBodyFail {
				r.Body = errReader{}
			}
			r = r.WithContext(tc.Context)
			r.RemoteAddr = "example.com:443"

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

func addWRPEventStreamDecoderInput() string {
	return `
		{
			"registered_from_address": "example.com:443",
			"config": {
				"url": "example.com:443",
				"content_type": "application/json",
				"secret": "superSecretXYZ"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "example.com",
			"duration": "0s",
			"until": "2021-01-02T15:04:10Z"
		}
	`
}
func addWRPEventStreamDecoderDurationInput() string {
	return `
		{
			"registered_from_address": "example.com:443",
			"config": {
				"url": "example.com:443",
				"content_type": "application/json",
				"secret": "superSecretXYZ"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "example.com",
			"duration": "300s"
		}
	`
}

func addWRPEventStreamDecoderUnmarshalingErrorInput(duration bool) string {
	if duration {
		return `
		{
			"config": {
				"url": "example.com:443",
				"content_type": "application/json",
				"secret": "superSecretXYZ"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "example.com",
			"duration": "hehe",
			"until": "2021-01-02T15:04:10Z"
		}
	`
	}
	return `
		{
			"config": {
				"url": "example.com:443",
				"content_type": 5,
				"secret": "superSecretXYZ"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "example.com",
			"duration": 0s,
			"until": "2021-01-02T15:04:10Z"
		}
	`
}

func addWRPEventStreamDecoderOutput(withPIDs bool) *addWRPEventStreamRequest {
	if withPIDs {
		return &addWRPEventStreamRequest{
			owner: "owner-from-auth",
			internalWebook: &schema.ManifestV1{
				// nolint:staticcheck
				Registration: webhook.RegistrationV1{
					Address: "example.com:443",
					// nolint:staticcheck
					Config: webhook.DeliveryConfig{
						ReceiverURL: "example.com:443",
						ContentType: "application/json",
						Secret:      "superSecretXYZ",
					},
					Events: []string{"online"},
					Matcher: webhook.MetadataMatcherConfig{
						DeviceID: []string{"mac:aabbccddee.*"},
					},
					FailureURL: "example.com",
					Duration:   webhook.CustomDuration(0 * time.Second),
					Until:      getRefTime().Add(10 * time.Second),
				},
				PartnerIDs: []string{"comcast"},
			},
		}
	}
	return &addWRPEventStreamRequest{
		owner: "owner-from-auth",
		internalWebook: &schema.ManifestV1{
			// nolint:staticcheck
			Registration: webhook.RegistrationV1{
				Address: "example.com:443",
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "example.com",
				Duration:   webhook.CustomDuration(0 * time.Second),
				Until:      getRefTime().Add(10 * time.Second),
			},
			PartnerIDs: []string{},
		},
	}
}
func addWRPEventStreamDecoderDurationOutput(withPIDs bool) *addWRPEventStreamRequest {
	if withPIDs {
		return &addWRPEventStreamRequest{
			owner: "owner-from-auth",
			internalWebook: &schema.ManifestV1{
				// nolint:staticcheck
				Registration: webhook.RegistrationV1{
					Address: "example.com:443",
					// nolint:staticcheck
					Config: webhook.DeliveryConfig{
						ReceiverURL: "example.com:443",
						ContentType: "application/json",
						Secret:      "superSecretXYZ",
					},
					Events: []string{"online"},
					Matcher: webhook.MetadataMatcherConfig{
						DeviceID: []string{"mac:aabbccddee.*"},
					},
					FailureURL: "example.com",
					Duration:   webhook.CustomDuration(5 * time.Minute),
					Until:      getRefTime().Add(5 * time.Minute),
				},
				PartnerIDs: []string{"comcast"},
			},
		}
	}
	return &addWRPEventStreamRequest{
		owner: "owner-from-auth",
		internalWebook: &schema.ManifestV1{
			// nolint:staticcheck
			Registration: webhook.RegistrationV1{
				Address: "example.com:443",
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "example.com",
				Duration:   webhook.CustomDuration(5 * time.Minute),
				Until:      getRefTime().Add(5 * time.Minute),
			},
			PartnerIDs: []string{},
		},
	}
}

func encodeGetAllInput() []schema.Manifest {
	return []schema.Manifest{
		&schema.ManifestV1{
			// nolint:staticcheck
			Registration: webhook.RegistrationV1{
				Address: "example.com:443",
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: struct {
					DeviceID []string `json:"device_id"`
				}{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "example.com",
				Duration:   webhook.CustomDuration(0 * time.Second),
				Until:      getRefTime().Add(10 * time.Second),
			},
			PartnerIDs: []string{"comcast"},
		},
		&schema.ManifestV1{
			// nolint:staticcheck
			Registration: webhook.RegistrationV1{
				Address: "example.com:443",
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ContentType: "application/json",
					ReceiverURL: "example.com:443",
					Secret:      "doNotShare:e=mc^2",
				},
				Events: []string{"online"},
				Matcher: struct {
					DeviceID []string `json:"device_id"`
				}{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "example.com",
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
			"registered_from_address": "example.com:443",
			"config": {
				"url": "example.com:443",
				"content_type": "application/json",
				"secret": "<obfuscated>"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "example.com",
			"duration": "0s",
			"until": "2021-01-02T15:04:10Z"
		},
		{
			"registered_from_address": "example.com:443",
			"config": {
				"url": "example.com:443",
				"content_type": "application/json",
				"secret": "<obfuscated>"
			},
			"events": ["online"],
			"matcher": {
				"device_id": ["mac:aabbccddee.*"]
			},
			"failure_url": "example.com",
			"duration": "0s",
			"until": "2021-01-02T15:04:20Z"
		}
	]
	`
}

func TestSetWRPEventStreamDefaults(t *testing.T) {
	tcs := []struct {
		desc string
		// nolint:staticcheck
		registration *webhook.RegistrationV1
		remoteAddr   string
		// nolint:staticcheck
		expectedRegistration *webhook.RegistrationV1
	}{
		{
			desc: "No Until, Address, or DeviceID",
			// nolint:staticcheck
			registration: &webhook.RegistrationV1{
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
				},
				Events:   []string{"online", "offline"},
				Matcher:  webhook.MetadataMatcherConfig{},
				Duration: webhook.CustomDuration(5 * time.Minute),
			},
			remoteAddr: "example.com:443",
			// nolint:staticcheck
			expectedRegistration: &webhook.RegistrationV1{
				Address: "example.com:443",
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
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
			// nolint:staticcheck
			registration: &webhook.RegistrationV1{
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
				},
				Events:   []string{"online", "offline"},
				Matcher:  webhook.MetadataMatcherConfig{},
				Duration: webhook.CustomDuration(5 * time.Minute),
			},
			// nolint:staticcheck
			expectedRegistration: &webhook.RegistrationV1{
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
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
			// nolint:staticcheck
			registration: &webhook.RegistrationV1{
				Address: "example.com:443",
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
				},
				Events: []string{"online", "offline"},
				Matcher: webhook.MetadataMatcherConfig{
					DeviceID: []string{".*"},
				},
				Duration: webhook.CustomDuration(5 * time.Minute),
				Until:    mockNow().Add(5 * time.Minute),
			},
			remoteAddr: "example.com:443",
			// nolint:staticcheck
			expectedRegistration: &webhook.RegistrationV1{
				Address: "example.com:443",
				// nolint:staticcheck
				Config: webhook.DeliveryConfig{
					ReceiverURL: "example.com:443",
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
			v := wrpEventStreamValidator{
				now: mockNow,
			}
			v.setV1Defaults(tc.registration, tc.remoteAddr)
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
