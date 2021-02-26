/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
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
	"github.com/xmidt-org/argus/model"
)

type PushReader interface {
	Pusher
	Reader
}

type Pusher interface {
	// Push applies user configurable for registering an item returning the id
	// i.e. updated the storage with said item.
	Push(item model.Item, owner string) (string, error)

	// Remove will remove the item from the store
	Remove(id string, owner string) (model.Item, error)
}

type Reader interface {
	// GeItems will return all the current items or an error.
	GetItems(owner string) ([]model.Item, error)
}
