package ancla

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

func newAddWebhookEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(*addWebhookRequest)

		return nil, s.Add(r.owner, r.webhook)
	}
}

func newGetAllWebhooksEndpoint(s Service) endpoint.Endpoint {
	return func(_ context.Context, _ interface{}) (interface{}, error) {
		return s.AllWebhooks()
	}
}
