package ancla

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAddWebhookEndpoint(t *testing.T) {
	assert := assert.New(t)
	m := new(mockService)
	endpoint := newAddWebhookEndpoint(m)
	input := &addWebhookRequest{
		owner:   "owner-val",
		webhook: Webhook{},
	}

	errFake := errors.New("failed")
	m.On("Add", context.TODO(), "owner-val", input.webhook).Return(errFake)
	resp, err := endpoint(context.Background(), input)
	assert.Nil(resp)
	assert.Equal(errFake, err)
	m.AssertExpectations(t)
}

func TestGetAllWebhooksEndpoint(t *testing.T) {
	assert := assert.New(t)
	m := new(mockService)
	endpoint := newGetAllWebhooksEndpoint(m)

	respFake := []Webhook{}
	m.On("AllWebhooks", context.TODO()).Return(respFake, nil)
	resp, err := endpoint(context.Background(), nil)
	assert.Nil(err)
	assert.Equal(respFake, resp)
	m.AssertExpectations(t)
}
