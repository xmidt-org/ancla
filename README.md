# ancla

ancla keeps the list of **wrpEventStream registrations** for XMiDT: the
subscriptions that tell caduceus where to deliver events.  It stores them in
[Argus](https://github.com/xmidt-org/argus), accepts and validates new ones
over HTTP, and hands the current list to anything that wants to follow it.

[![Build Status](https://github.com/xmidt-org/ancla/actions/workflows/ci.yml/badge.svg)](https://github.com/xmidt-org/ancla/actions/workflows/ci.yml)
[![codecov.io](http://codecov.io/github/xmidt-org/ancla/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/ancla?branch=main)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/ancla/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/xmidt-org/ancla.svg)](https://github.com/xmidt-org/ancla/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/xmidt-org/ancla)](https://pkg.go.dev/github.com/xmidt-org/ancla)


## Where it fits

| Service | Uses ancla to |
|---|---|
| [tr1d1um](https://github.com/xmidt-org/tr1d1um) | accept registrations (`POST`) and list them (`GET`) |
| [caduceus](https://github.com/xmidt-org/caduceus) | watch the stored list and deliver events to each registration |

Both talk to the same Argus bucket, so they must agree on how a registration
is stored.  See [Storage layouts](#storage-layouts), which matters when the
two run different versions.

## Packages

| Package | Purpose |
|---|---|
| `ancla` | `Registry` (`Add`, `GetAll`, `GetAllByOwner`), the HTTP handlers, and `Watch` for following updates |
| `ancla/anclafx` | [fx](https://github.com/uber-go/fx) module wiring everything together |
| `ancla/chrysom` | Argus client: `GetItems`, `PushItem`, `RemoveItem`, plus a polling listener |
| `ancla/schema` | the registration types and how they are converted to and from stored items |
| `ancla/auth` | request decorators and the principal and partner IDs carried in a context |
| `ancla/model` | the item stored in Argus |

## Registrations

Two registration schemas are accepted.  The body decides which: a request with
`canonical_name`, `webhooks` or `kafkas` is v2, anything else is v1.

- **v1** is the original webhook registration (`config.url`, `events`,
  `duration`/`until`, and so on).  ancla fills in defaults such as the
  device matcher and the expiry, and records the address of the caller.
- **v2** has a `canonical_name`, one or more `webhooks` or `kafkas`, and a
  single `expires` time in place of `duration`.

The schemas themselves live in
[webhook-schema](https://github.com/xmidt-org/webhook-schema).  Validation
(URL rules, regular expressions, lifetime limits) is configured with
`Config.Validation`.  Secrets are replaced with `<obfuscated>` when
registrations are listed.

## Usage

### With fx

```go
app := fx.New(
	anclafx.Provide(),
	// provide: a touchstone Factory, chrysom.ClientOptions (at least a
	// Bucket), chrysom.ListenerOptions, and any Watch values you want
	// called on every update.
)
```

This provides a `*ancla.Registry`, the Argus client, and a listener that polls
Argus and calls every `Watch`.

### Without fx

```go
client, err := chrysom.NewBasicClient(
	chrysom.StoreBaseURL("https://argus.example.com"),
	chrysom.Bucket("webhooks"),
)
if err != nil {
	return err
}

registry := ancla.NewRegistry(client)

err = registry.Add(ctx, owner, &schema.ManifestV1{
	PartnerIDs:   partners,
	Registration: registration, // a webhook.RegistrationV1
})

manifests, err := registry.GetAll(ctx) // every owner's; each is a *schema.ManifestV1 or *schema.ManifestV2

mine, err := registry.GetAllByOwner(ctx, owner) // only owner's; an empty owner is an error
```

`ancla.NewAddWRPEventStreamHandler` and `ancla.NewGetAllWRPEventStreamsHandler`
return ready-made `http.Handler`s for the two operations.  They read the
principal and partner IDs that your authentication middleware stored with
`auth.SetPrincipal` and `auth.SetPartnerIDs`.  Set `HandlerConfig.FilterByOwner`
to have the list handler return only the caller's registrations; it answers
401 if the request has no principal, and never falls back to listing everyone's.

## Storage layouts

A registration is stored as a JSON item under a key that names its layout.
`GetAll` and the listener understand all of them, so a reader never needs to
know which version wrote an item.

| Registration | Key | Written by |
|---|---|---|
| v1 | `Webhook` | ancla v0.3.12, and v0.6.0 or later by default |
| v1 | `wrp_event_stream_schema_v1` | v0.4.0 to v0.5.6, and v0.6.0 or later with `OutputV1` |
| v2 | `wrp_event_stream_schema_v2` | v0.4.0 or later |

A v1 registration is written under `Webhook`, with the duration as integer
nanoseconds, so that **every** version can read it, including v0.3.12.  To
write the other layout, pass `ancla.WithOutputFormat(schema.OutputV1)` to
`NewRegistry`, or provide a `schema.OutputFormat` to fx.  The format only
affects v1; a v2 registration has one layout.

### Compatibility

- v0.3.12 cannot read a v2 registration.  Do not accept v2 registrations
  until every reader has been upgraded.
- **v0.4.0 through v0.5.6 are retracted.**  They wrote v1 registrations in a
  layout v0.3.12 readers cannot read.  Use v0.6.0 or later.
- Upgrade readers (caduceus) before, or together with, writers (tr1d1um) if
  you want to move to `OutputV1`.

## Maintenance Instructions

This repository uses `shared-go` for it's workflows.  [Here is documentation](https://github.com/xmidt-org/shared-go/#maintaining-a-repository-using-shared-go)
for how to best maintain this repository.

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/docs/community/code_of_conduct). 
By participating, you agree to this Code.

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
