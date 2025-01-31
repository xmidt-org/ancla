// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/xmidt-org/ancla/auth"
	"github.com/xmidt-org/ancla/model"
	"go.uber.org/zap"
)

// Request and Response Headers.
const (
	ItemOwnerHeaderKey  = "X-Xmidt-Owner"
	XmidtErrorHeaderKey = "X-Xmidt-Error"
)

var (
	ErrItemIDEmpty          = errors.New("item ID is required")
	ErrItemDataEmpty        = errors.New("data field in item is required")
	ErrAuthDecoratorFailure = errors.New("failed decorating auth header")
	ErrBadRequest           = errors.New("argus rejected the request as invalid")
)

var (
	ErrFailedAuthentication = errors.New("failed to authentication with argus")
	errNonSuccessResponse   = errors.New("argus responded with a non-success status code")
	errNewRequestFailure    = errors.New("failed creating an HTTP request")
	errDoRequestFailure     = errors.New("http client failed while sending request")
	errReadingBodyFailure   = errors.New("failed while reading http response body")
	errJSONUnmarshal        = errors.New("failed unmarshaling JSON response payload")
	errJSONMarshal          = errors.New("failed marshaling item as JSON payload")
)

// BasicClient is the client used to make requests to Argus.
type BasicClient struct {
	client       *http.Client
	auth         auth.Decorator
	storeBaseURL string
	storeAPIPath string
	bucket       string
	getLogger    func(context.Context) *zap.Logger
}

type response struct {
	Body             []byte
	ArgusErrorHeader string
	Code             int
}

const (
	storeV1APIPath   = "/api/v1/store"
	errWrappedFmt    = "%w: %s"
	errStatusCodeFmt = "%w: received status %v"
	errorHeaderKey   = "errorHeader"
)

var (
	defaultClientOptions = ClientOptions{
		// localhost defaults
		StoreBaseURL(""),
		StoreAPIPath(""),
		// Nop defaults
		HTTPClient(nil),
		GetClientLogger(nil),
		Auth(nil),
	}
)

// NewBasicClient creates a new BasicClient that can be used to
// make requests to Argus.
func NewBasicClient(opts ...ClientOption) (*BasicClient, error) {
	var client BasicClient

	opts = append(defaultClientOptions, ClientOptions(opts))
	opts = append(opts, clientValidator())

	return &client, ClientOptions(opts).apply(&client)
}

// GetItems fetches all items that belong to a given owner.
func (c *BasicClient) GetItems(ctx context.Context, owner string) (Items, error) {
	response, err := c.sendRequest(ctx, owner, http.MethodGet, fmt.Sprintf("%s/%s", c.storeBaseURL, c.bucket), nil)
	if err != nil {
		return nil, err
	}

	if response.Code != http.StatusOK {
		c.getLogger(ctx).Error("Argus responded with non-200 response for GetItems request",
			zap.Int("code", response.Code), zap.String(errorHeaderKey, response.ArgusErrorHeader))
		return nil, fmt.Errorf(errStatusCodeFmt, translateNonSuccessStatusCode(response.Code), response.Code)
	}

	var items Items

	err = json.Unmarshal(response.Body, &items)
	if err != nil {
		return nil, fmt.Errorf("GetItems: %w: %s", errJSONUnmarshal, err.Error())
	}

	return items, nil
}

// PushItem creates a new item if one doesn't already exist. If an item exists
// and the ownership matches, the item is simply updated.
func (c *BasicClient) PushItem(ctx context.Context, owner string, item model.Item) (PushResult, error) {
	err := validatePushItemInput(owner, item)
	if err != nil {
		return NilPushResult, err
	}

	data, err := json.Marshal(item)
	if err != nil {
		return NilPushResult, fmt.Errorf(errWrappedFmt, errJSONMarshal, err.Error())
	}

	response, err := c.sendRequest(ctx, owner, http.MethodPut, fmt.Sprintf("%s/%s/%s", c.storeBaseURL, c.bucket, item.ID), bytes.NewReader(data))
	if err != nil {
		return NilPushResult, err
	}

	if response.Code == http.StatusCreated {
		return CreatedPushResult, nil
	}

	if response.Code == http.StatusOK {
		return UpdatedPushResult, nil
	}

	c.getLogger(ctx).Error("Argus responded with a non-successful status code for a PushItem request",
		zap.Int("code", response.Code), zap.String(errorHeaderKey, response.ArgusErrorHeader))

	return NilPushResult, fmt.Errorf(errStatusCodeFmt, translateNonSuccessStatusCode(response.Code), response.Code)
}

// RemoveItem removes the item if it exists and returns the data associated to it.
func (c *BasicClient) RemoveItem(ctx context.Context, id, owner string) (model.Item, error) {
	if len(id) < 1 {
		return model.Item{}, ErrItemIDEmpty
	}

	resp, err := c.sendRequest(ctx, owner, http.MethodDelete, fmt.Sprintf("%s/%s/%s", c.storeBaseURL, c.bucket, id), nil)
	if err != nil {
		return model.Item{}, err
	}

	if resp.Code != http.StatusOK {
		c.getLogger(ctx).Error("Argus responded with a non-successful status code for a RemoveItem request",
			zap.Int("code", resp.Code), zap.String(errorHeaderKey, resp.ArgusErrorHeader))
		return model.Item{}, fmt.Errorf(errStatusCodeFmt, translateNonSuccessStatusCode(resp.Code), resp.Code)
	}

	var item model.Item
	err = json.Unmarshal(resp.Body, &item)
	if err != nil {
		return item, fmt.Errorf("RemoveItem: %w: %s", errJSONUnmarshal, err.Error())
	}
	return item, nil
}

func validatePushItemInput(_ string, item model.Item) error {
	if len(item.ID) < 1 {
		return ErrItemIDEmpty
	}

	if len(item.Data) < 1 {
		return ErrItemDataEmpty
	}

	return nil
}

func (c *BasicClient) sendRequest(ctx context.Context, owner, method, url string, body io.Reader) (response, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return response{}, fmt.Errorf(errWrappedFmt, errNewRequestFailure, err.Error())
	}

	if len(owner) > 0 {
		r.Header.Set(ItemOwnerHeaderKey, owner)
	}

	if c.auth != nil {
		if err := c.auth.Decorate(ctx, r); err != nil {
			return response{}, errors.Join(ErrAuthDecoratorFailure, err)
		}
	}

	resp, err := c.client.Do(r)
	if err != nil {
		return response{}, fmt.Errorf(errWrappedFmt, errDoRequestFailure, err.Error())
	}

	defer resp.Body.Close()

	sqResp := response{
		Code:             resp.StatusCode,
		ArgusErrorHeader: resp.Header.Get(XmidtErrorHeaderKey),
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return sqResp, fmt.Errorf(errWrappedFmt, errReadingBodyFailure, err.Error())
	}

	sqResp.Body = bodyBytes

	return sqResp, nil
}

// translateNonSuccessStatusCode returns as specific error
// for known Argus status codes.
func translateNonSuccessStatusCode(code int) error {
	switch code {
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrFailedAuthentication
	default:
		return errNonSuccessResponse
	}
}
