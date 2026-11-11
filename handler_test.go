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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/ancla/auth"
	"github.com/xmidt-org/ancla/chrysom"
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
			expectedBody: `{"message": "failed to add wrpEventStream to registry: boom"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockPushReader)
			if tc.expectAdd {
				// nolint:typecheck
				m.On("PushItem", mock.Anything, testOwner, mock.Anything).Return(chrysom.CreatedPushResult, tc.serviceErr)
			}

			h := NewAddWRPEventStreamHandler(NewRegistry(m), testHandlerConfig())

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
		fetchErr     error
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
			desc:         "fetch failure is an internal error",
			fetchErr:     errors.New("boom"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"message": "failed to fetch wrpEventStreams: boom"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			// Fixed items, in order: the fixtures share a URL, which a keyed
			// store would collapse into one.
			items := chrysom.Items{}
			for _, m := range tc.manifests {
				item, err := schema.SchemaToItem(time.Now, m)
				assert.NoError(err)
				items = append(items, item)
			}
			store := new(mockPushReader)
			// nolint:typecheck
			store.On("GetItems", mock.Anything, "").Return(items, tc.fetchErr)
			reg := NewRegistry(store)

			h := NewGetAllWRPEventStreamsHandler(reg, testHandlerConfig())

			r := httptest.NewRequest(http.MethodGet, "/hooks", nil)
			recorder := httptest.NewRecorder()

			h.ServeHTTP(recorder, r)

			assert.Equal(tc.expectedCode, recorder.Code)
			assert.Equal(testContentType, recorder.Header().Get("Content-Type"))
			assert.JSONEq(tc.expectedBody, recorder.Body.String())
		})
	}
}

func TestGetAllHandlerFilterByOwner(t *testing.T) {
	store := &memStore{}
	reg := NewRegistry(store)
	ctx := context.Background()
	require.NoError(t, reg.Add(ctx, "alice", v1Manifest("http://alice.example.com")))
	require.NoError(t, reg.Add(ctx, "bob", v1Manifest("http://bob.example.com")))

	list := func(t *testing.T, filter bool, ctx context.Context) (int, string) {
		t.Helper()
		cfg := testHandlerConfig()
		cfg.FilterByOwner = filter
		r := httptest.NewRequest(http.MethodGet, "/hooks", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		NewGetAllWRPEventStreamsHandler(reg, cfg).ServeHTTP(w, r)
		return w.Code, w.Body.String()
	}

	t.Run("off lists everyone's, as before", func(t *testing.T) {
		code, body := list(t, false, auth.SetPrincipal(ctx, "alice"))
		assert.Equal(t, http.StatusOK, code)
		assert.Contains(t, body, "alice.example.com")
		assert.Contains(t, body, "bob.example.com")
	})

	t.Run("off needs no principal", func(t *testing.T) {
		code, _ := list(t, false, ctx)
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("on lists only the caller's", func(t *testing.T) {
		code, body := list(t, true, auth.SetPrincipal(ctx, "alice"))
		assert.Equal(t, http.StatusOK, code)
		assert.Contains(t, body, "alice.example.com")
		assert.NotContains(t, body, "bob.example.com")
	})

	t.Run("on with no principal fails closed", func(t *testing.T) {
		code, body := list(t, true, ctx)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.NotContains(t, body, "example.com")
	})

	t.Run("on with an empty principal fails closed", func(t *testing.T) {
		code, body := list(t, true, auth.SetPrincipal(ctx, ""))
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.NotContains(t, body, "example.com")
	})
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
