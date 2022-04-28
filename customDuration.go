/**
 * Copyright 2022 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ancla

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

type InvalidDurationError struct {
	Value string
}

func (ide *InvalidDurationError) Error() string {
	var o strings.Builder
	o.WriteString("duration must be of type int or string (ex:'5m'); Invalid value: ")
	o.WriteString(ide.Value)
	return o.String()
}

type CustomDuration time.Duration

func (cd CustomDuration) String() string {
	return time.Duration(cd).String()
}

func (cd CustomDuration) MarshalJSON() ([]byte, error) {
	d := bytes.NewBuffer(nil)
	d.WriteByte('"')
	d.WriteString(cd.String())
	d.WriteByte('"')
	return d.Bytes(), nil
}

func (cd *CustomDuration) UnmarshalJSON(b []byte) (err error) {
	if b[0] == '"' {
		var d time.Duration
		d, err = time.ParseDuration(string(b[1 : len(b)-1]))
		if err == nil {
			*cd = CustomDuration(d)
			return
		}
	}

	var d int64
	d, err = strconv.ParseInt(string(b), 10, 64)
	if err == nil {
		*cd = CustomDuration(time.Duration(d) * time.Second)
		return
	}

	err = &InvalidDurationError{
		Value: string(b),
	}

	return
}
