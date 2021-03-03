package ancla

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebhookListSizeWatch(t *testing.T) {
	require := require.New(t)
	gauge := new(mockGauge)
	watch := webhookListSizeWatch(gauge)
	require.NotNil(watch)
	gauge.On("Set", float64(2))
	watch.Update([]Webhook{{}, {}})
	gauge.AssertExpectations(t)
}
