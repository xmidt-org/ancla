// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/ancla/schema"
)

const errFmt = "%w: %v"

var (
	errNonSuccessPushResult           = errors.New("got a push result but was not of success type")
	errFailedWRPEventStreamPush       = errors.New("failed to add wrpEventStream to registry")
	errFailedWRPEventStreamConversion = errors.New("failed to convert wrpEventStream to argus item")
	errFailedItemConversion           = errors.New("failed to convert argus item to wrpEventStream")
	errFailedWRPEventStreamsFetch     = errors.New("failed to fetch wrpEventStreams")
)

// Service describes the core operations around wrpEventStream subscriptions.
type Service interface {
	// Add adds the given owned wrpEventStream to the current list of wrpEventStreams. If the operation
	// succeeds, a non-nil error is returned.
	Add(ctx context.Context, owner string, iw schema.RegistryManifest) error

	// GetAll lists all the current registered wrpEventStreams.
	GetAll(ctx context.Context) ([]schema.RegistryManifest, error)
}

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

type service struct {
	// argus is the Argus database client.
	argus chrysom.PushReader
	now   func() time.Time
}

// Add adds the given owned wrpEventStream to the current list of wrpEventStreams. If the operation
// succeeds, a non-nil error is returned.
func (s *service) Add(ctx context.Context, owner string, iw schema.RegistryManifest) error {
	item, err := schema.SchemaToItem(s.now, iw)
	if err != nil {
		return fmt.Errorf(errFmt, errFailedWRPEventStreamConversion, err)
	}
	result, err := s.argus.PushItem(ctx, owner, item)
	if err != nil {
		return fmt.Errorf(errFmt, errFailedWRPEventStreamPush, err)
	}

	if result == chrysom.CreatedPushResult || result == chrysom.UpdatedPushResult {
		return nil
	}
	return fmt.Errorf("%w: %s", errNonSuccessPushResult, result)
}

// GetAll returns all wrpEventStreams found on the configured wrpEventStreams partition
// of Argus.
func (s *service) GetAll(ctx context.Context) ([]schema.RegistryManifest, error) {
	items, err := s.argus.GetItems(ctx, "")
	if err != nil {
		return nil, fmt.Errorf(errFmt, errFailedWRPEventStreamsFetch, err)
	}

	iws := make([]schema.RegistryManifest, len(items))

	for i, item := range items {
		wrpEventStream, err := schema.ItemToSchema(item)
		if err != nil {
			return nil, fmt.Errorf(errFmt, errFailedItemConversion, err)
		}
		iws[i] = wrpEventStream
	}

	return iws, nil
}

// NewService returns an ancla client used to interact with an Argus database.
func NewService(client chrysom.PushReader) *service {
	return &service{
		argus: client,
		now:   time.Now,
	}
}
