// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJSON(t *testing.T) {
	type test struct {
		Duration CustomDuration
	}
	tests := []struct {
		description      string
		input            []byte
		expectedDuration CustomDuration
		errExpected      bool
	}{
		{
			description:      "Int success",
			input:            []byte(`{"duration":50}`),
			expectedDuration: CustomDuration(50 * time.Second),
		},
		{
			description:      "String success",
			input:            []byte(`{"duration":"5m"}`),
			expectedDuration: CustomDuration(5 * time.Minute),
		},
		{
			description: "String failure",
			input:       []byte(`{"duration":"2r"}`),
			errExpected: true,
		},
		{
			description: "Object failure",
			input:       []byte(`{"duration":{"key":"val"}}`),
			errExpected: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			cd := test{}
			err := json.Unmarshal(tc.input, &cd)
			assert.Equal(tc.expectedDuration, cd.Duration)
			if !tc.errExpected {
				assert.NoError(err)
				return
			}
			assert.Error(err)
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	type test struct {
		Duration CustomDuration
	}
	tests := []struct {
		description    string
		input          test
		expectedOutput []byte
		errExpected    bool
	}{
		{
			description:    "Int success",
			input:          test{Duration: CustomDuration(50 * time.Second)},
			expectedOutput: []byte(`{"Duration":"50s"}`),
		},
		{
			description:    "String success",
			input:          test{Duration: CustomDuration(5 * time.Minute)},
			expectedOutput: []byte(`{"Duration":"5m0s"}`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			output, err := json.Marshal(tc.input)
			assert.Equal(tc.expectedOutput, output)
			if !tc.errExpected {
				assert.NoError(err)
				return
			}
			assert.Error(err)
		})
	}

}
