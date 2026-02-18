/*
 * Copyright 2026 Simon Emms <simon@simonemms.com>
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
 */

package wf

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

type Activities struct{}

func (a *Activities) Create(ctx context.Context, heartbeatTimeout time.Duration) error {
	// Start a heartbeat
	_, cancel := ExecuteEvery(ctx, heartbeatTimeout/2, func(hctx context.Context) {
		activity.RecordHeartbeat(hctx)
	})
	defer cancel()

	fmt.Println("---")
	fmt.Println("hello, I am an activity")
	fmt.Println("---")

	time.Sleep(time.Second * 10)

	return nil
}

func NewActivities() *Activities {
	return &Activities{}
}
