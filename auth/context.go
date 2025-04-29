// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import "context"

type PartnerIDsKey struct{}

type PrincipalKey struct{}

// SetPrincipal adds the security principal to the context given, e.g. the user name or client id.
func SetPrincipal(ctx context.Context, p string) context.Context {
	return context.WithValue(ctx, PrincipalKey{}, p)
}

// GetPrincipal gets the security principal from the context provided.
func GetPrincipal(ctx context.Context) (p string, ok bool) {
	p, ok = ctx.Value(PrincipalKey{}).(string)
	return
}

// SetPartnerIDs adds the list of partner IDs to the context given.
func SetPartnerIDs(ctx context.Context, ids []string) context.Context {
	return context.WithValue(ctx, PartnerIDsKey{}, ids)
}

// GetPartnerIDs gets the list of partner IDs from the context provided.
func GetPartnerIDs(ctx context.Context) (ids []string, ok bool) {
	ids, ok = ctx.Value(PartnerIDsKey{}).([]string)
	return
}
