// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/ancla/auth"
	"github.com/xmidt-org/ancla/schema"
	"go.uber.org/zap"
)

func testHandlerConfig() HandlerConfig {
	return HandlerConfig{
		GetLogger: func(context.Context) *zap.Logger {
			return zap.NewNop()
		},
	}
}

func TestNewAddWRPEventStreamHandler(t *testing.T) {
	tcs := []struct {
		desc         string
		payload      string
		ctx          context.Context
		serviceErr   error
		expectAdd    bool
		expectedCode int
		expectedBody string
	}{
		{
			desc:         testSuccess,
			payload:      addWRPEventStreamDecoderInput(),
			ctx:          auth.SetPartnerIDs(auth.SetPrincipal(context.Background(), testOwner), []string{testPartnerID}),
			expectAdd:    true,
			expectedCode: http.StatusOK,
			expectedBody: `{"message": "Success"}`,
		},
		{
			desc:         "decoder failure is a bad request",
			payload:      `not json`,
			ctx:          auth.SetPartnerIDs(auth.SetPrincipal(context.Background(), testOwner), []string{testPartnerID}),
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "missing partner ids is a bad request",
			payload:      addWRPEventStreamDecoderInput(),
			ctx:          auth.SetPrincipal(context.Background(), testOwner),
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"message": "failed getting partnerIDs: unable to retrieve PartnerIDs"}`,
		},
		{
			desc:         "service failure is an internal error",
			payload:      addWRPEventStreamDecoderInput(),
			ctx:          auth.SetPartnerIDs(auth.SetPrincipal(context.Background(), testOwner), []string{testPartnerID}),
			serviceErr:   errors.New("boom"),
			expectAdd:    true,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"message": "boom"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockService)
			if tc.expectAdd {
				// nolint:typecheck
				m.On("Add", mock.Anything, testOwner, mock.Anything).Return(tc.serviceErr)
			}

			h := NewAddWRPEventStreamHandler(m, testHandlerConfig())

			r := httptest.NewRequest(http.MethodPost, "/hooks", bytes.NewBufferString(tc.payload))
			r = r.WithContext(tc.ctx)
			recorder := httptest.NewRecorder()

			h.ServeHTTP(recorder, r)

			assert.Equal(tc.expectedCode, recorder.Code)
			assert.Equal(testContentType, recorder.Header().Get("Content-Type"))
			if tc.expectedBody != "" {
				assert.JSONEq(tc.expectedBody, recorder.Body.String())
			}
			// nolint:typecheck
			m.AssertExpectations(t)
		})
	}
}

func TestNewGetAllWRPEventStreamsHandler(t *testing.T) {
	tcs := []struct {
		desc         string
		manifests    []schema.Manifest
		serviceErr   error
		expectedCode int
		expectedBody string
	}{
		{
			desc:         testSuccess,
			manifests:    encodeGetAllInput(),
			expectedCode: http.StatusOK,
			expectedBody: encodeGetAllOutput(),
		},
		{
			desc:         "empty",
			expectedCode: http.StatusOK,
			expectedBody: `[]`,
		},
		{
			desc:         "service failure is an internal error",
			serviceErr:   errors.New("boom"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"message": "boom"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockService)
			// nolint:typecheck
			m.On("GetAll", mock.Anything).Return(tc.manifests, tc.serviceErr)

			h := NewGetAllWRPEventStreamsHandler(m, testHandlerConfig())

			r := httptest.NewRequest(http.MethodGet, "/hooks", nil)
			recorder := httptest.NewRecorder()

			h.ServeHTTP(recorder, r)

			assert.Equal(tc.expectedCode, recorder.Code)
			assert.Equal(testContentType, recorder.Header().Get("Content-Type"))
			assert.JSONEq(tc.expectedBody, recorder.Body.String())
			// nolint:typecheck
			m.AssertExpectations(t)
		})
	}
}

func TestServerEncoderFailure(t *testing.T) {
	assert := assert.New(t)
	errEncode := errors.New("encode failed")

	h := newServer(
		func(context.Context, any) (any, error) { return "ok", nil },
		nopRequestDecoder,
		func(context.Context, http.ResponseWriter, any) error { return errEncode },
		errorEncoder(testHandlerConfig().GetLogger),
	)

	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/hooks", nil))

	assert.Equal(http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(`{"message": "encode failed"}`, recorder.Body.String())
}
