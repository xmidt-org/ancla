// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

// Names
const (
	WebhookListSizeGaugeName = "webhook_list_size"
	WebhookListSizeGaugeHelp = "Size of the current list of webhooks."
	PollsTotalCounterName    = "chrysom_polls_total"
	PollsTotalCounterHelp    = "Counter for the number of polls (and their success/failure outcomes) to fetch new items."
)

// Labels
const (
	OutcomeLabel = "outcome"
)

// Label Values
const (
	SuccessOutcome = "success"
	FailureOutcome = "failure"
)
