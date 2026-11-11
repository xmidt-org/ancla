// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	webhook "github.com/xmidt-org/webhook-schema"
)

// A v1 registration is stored in the layout ancla v0.3.12 used ("Webhook"),
// so that readers running v0.3.12 and later can all read it.  Items written
// by the intervening releases under "wrp_event_stream_schema_v1" are still
// read.  v2 registrations have no v0.3.12 equivalent and are stored under
// "wrp_event_stream_schema_v2".

// OutputFormat selects how a v1 registration is laid out when it is written
// to the store.  Reading is never affected: every layout is always readable.
// v2 registrations are always stored under "wrp_event_stream_schema_v2".
type OutputFormat int

const (
	// OutputLegacy stores a v1 registration under "Webhook", exactly as ancla
	// v0.3.12 did, so readers of any version can read it.  This is the default.
	OutputLegacy OutputFormat = iota

	// OutputV1 stores a v1 registration under "wrp_event_stream_schema_v1", as
	// the releases after v0.3.12 did.  v0.3.12 readers cannot read it.
	OutputV1
)

var errUnknownOutputFormat = errors.New("unknown output format")

// String implements fmt.Stringer.
func (f OutputFormat) String() string {
	switch f {
	case OutputLegacy:
		return "legacy"
	case OutputV1:
		return "v1"
	default:
		return fmt.Sprintf("OutputFormat(%d)", int(f))
	}
}

// UnmarshalText allows an OutputFormat to be set from configuration.  The
// empty string selects OutputLegacy.
func (f *OutputFormat) UnmarshalText(text []byte) error {
	switch string(text) {
	case "", "legacy":
		*f = OutputLegacy
	case "v1":
		*f = OutputV1
	default:
		return fmt.Errorf("%w: %q", errUnknownOutputFormat, string(text))
	}

	return nil
}

// legacyItem is the exact layout ancla v0.3.12 stored (InternalWebhook).
type legacyItem struct {
	PartnerIDs []string
	Webhook    legacyWebhook
}

// legacyWebhook is the v0.3.12 Webhook, also the payload read from items that
// hold a v1 registration under "wrp_event_stream_schema_v1".
// nolint:staticcheck
type legacyWebhook struct {
	Address    string                        `json:"registered_from_address"`
	Config     webhook.DeliveryConfig        `json:"config"`
	FailureURL string                        `json:"failure_url"`
	Events     []string                      `json:"events"`
	Matcher    webhook.MetadataMatcherConfig `json:"matcher,omitempty"`
	Duration   storedDuration                `json:"duration"`
	Until      time.Time                     `json:"until"`
}

// nolint:staticcheck
func toLegacy(m *ManifestV1) legacyItem {
	r := m.Registration
	return legacyItem{
		PartnerIDs: m.PartnerIDs,
		Webhook: legacyWebhook{
			Address:    r.Address,
			Config:     r.Config,
			FailureURL: r.FailureURL,
			Events:     r.Events,
			Matcher:    r.Matcher,
			Duration:   storedDuration(r.Duration),
			Until:      r.Until,
		},
	}
}

// nolint:staticcheck
func (l legacyItem) manifest() *ManifestV1 {
	w := l.Webhook
	return &ManifestV1{
		PartnerIDs: l.PartnerIDs,
		Registration: webhook.RegistrationV1{
			Address:    w.Address,
			Config:     w.Config,
			FailureURL: w.FailureURL,
			Events:     w.Events,
			Matcher:    w.Matcher,
			Duration:   webhook.CustomDuration(w.Duration),
			Until:      w.Until,
		},
	}
}

// storedDuration is a duration as it appears in a stored item.  It always
// encodes as integer nanoseconds, as v0.3.12 did.  It decodes integer
// nanoseconds or a duration string such as "5m".
//
// webhook.CustomDuration must not be used for stored items: it decodes
// integers as seconds, which is right for a registration request but wrong
// for anything ancla wrote, since ancla never stores integer seconds.
type storedDuration time.Duration

func (d storedDuration) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(d), 10)), nil
}

func (d *storedDuration) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		return nil
	}

	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		v, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		*d = storedDuration(v)
		return nil
	}

	n, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return fmt.Errorf("duration must be integer nanoseconds or a duration string: %s", b)
	}
	*d = storedDuration(n)
	return nil
}

// interimItem is an item holding a v1 registration under
// "wrp_event_stream_schema_v1", as written by releases after v0.3.12.
type interimItem struct {
	PartnerIDs   []string
	Registration legacyWebhook `json:"wrp_event_stream_schema_v1"`
}

func (i interimItem) manifest() *ManifestV1 {
	return legacyItem{PartnerIDs: i.PartnerIDs, Webhook: i.Registration}.manifest()
}
