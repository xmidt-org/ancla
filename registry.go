// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/ancla/schema"
	"github.com/xmidt-org/httpaux/erraux"
)

const errFmt = "%w: %v"

// ErrNoOwner is returned by GetAllByOwner when it is given an empty owner.
// Listing with an empty owner would return every owner's wrpEventStreams, so
// it fails instead.
var ErrNoOwner = errors.New("an owner is required to list wrpEventStreams")

var (
	errNonSuccessPushResult           = errors.New("got a push result but was not of success type")
	errFailedWRPEventStreamPush       = errors.New("failed to add wrpEventStream to registry")
	errFailedWRPEventStreamConversion = errors.New("failed to convert wrpEventStream to argus item")
	errFailedItemConversion           = errors.New("failed to convert argus item to wrpEventStream")
	errFailedWRPEventStreamsFetch     = errors.New("failed to fetch wrpEventStreams")
)

// Config contains information needed to initialize the Argus database client.
type Config struct {
	// DisablePartnerIDs, if true, will allow wrpEventStreams to register without
	// checking the validity of the partnerIDs in the request.
	DisablePartnerIDs bool

	// Validation provides options for validating the wrpEventStream's URL and TTL
	// related fields. Some validation happens regardless of the configuration:
	// URLs must be a valid URL structure, the Matcher.DeviceID values must
	// compile into regular expressions, and the Events field must have at
	// least one value and all values must compile into regular expressions.
	Validation schema.SchemaURLValidatorConfig
}

// Registry stores wrpEventStream registrations in Argus and lists them.
// Create one with NewRegistry.
type Registry struct {
	// argus is the Argus database client.
	argus  chrysom.PushReader
	now    func() time.Time
	format schema.OutputFormat
}

// Option configures a Registry.
type Option func(*Registry)

// WithOutputFormat selects the layout used to store v1 registrations.  The
// default is schema.OutputLegacy, which every version of ancla can read.
// Reads always understand every layout.
func WithOutputFormat(f schema.OutputFormat) Option {
	return func(r *Registry) { r.format = f }
}

// NewRegistry returns a Registry that stores registrations using the given
// Argus client.
func NewRegistry(client chrysom.PushReader, opts ...Option) *Registry {
	r := &Registry{
		argus: client,
		now:   time.Now,
	}
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Add adds the given owned wrpEventStream to the current list of
// wrpEventStreams.  A nil error means it was stored.
func (r *Registry) Add(ctx context.Context, owner string, manifest schema.Manifest) error {
	item, err := schema.SchemaToItemFormat(r.now, manifest, r.format)
	if err != nil {
		return fmt.Errorf(errFmt, errFailedWRPEventStreamConversion, err)
	}
	result, err := r.argus.PushItem(ctx, owner, item)
	if err != nil {
		return fmt.Errorf(errFmt, errFailedWRPEventStreamPush, err)
	}

	if result == chrysom.CreatedPushResult || result == chrysom.UpdatedPushResult {
		return nil
	}
	return fmt.Errorf("%w: %s", errNonSuccessPushResult, result)
}

// GetAll returns every owner's wrpEventStreams.  To list only one owner's, use
// GetAllByOwner.
func (r *Registry) GetAll(ctx context.Context) ([]schema.Manifest, error) {
	return r.list(ctx, "")
}

// GetAllByOwner returns the wrpEventStreams that belong to owner.  An empty
// owner is an error (ErrNoOwner, with status 401) and never lists everything.
func (r *Registry) GetAllByOwner(ctx context.Context, owner string) ([]schema.Manifest, error) {
	if owner == "" {
		return nil, &erraux.Error{Err: ErrNoOwner, Code: http.StatusUnauthorized}
	}

	return r.list(ctx, owner)
}

// list fetches the items for owner, or every owner's if owner is empty.
func (r *Registry) list(ctx context.Context, owner string) ([]schema.Manifest, error) {
	items, err := r.argus.GetItems(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf(errFmt, errFailedWRPEventStreamsFetch, err)
	}

	manifests := make([]schema.Manifest, len(items))
	for i, item := range items {
		m, err := schema.ItemToSchema(item)
		if err != nil {
			return nil, fmt.Errorf(errFmt, errFailedItemConversion, err)
		}
		manifests[i] = m
	}

	return manifests, nil
}
