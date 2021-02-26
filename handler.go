package ancla

import (
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
)

func NewAddWebhookHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newAddWebhookEndpoint(s),
		decodeAddWebhookRequest,
		encodeAddWebhookResponse,
		kithttp.ServerErrorEncoder(errorEncoder),
	)
}

func NewGetAllWebhooksHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newGetAllWebhooksEndpoint(s),
		kithttp.NopRequestDecoder,
		encodeGetAllWebhooksResponse,
		kithttp.ServerErrorEncoder(errorEncoder),
	)
}
