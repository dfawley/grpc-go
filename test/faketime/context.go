/*
 *
 * Copyright 2018 gRPC authors.
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

package faketime

import (
	"context"
	"time"
)

type dc struct {
	context.Context
	Timer
}

func (d *dc) Deadline() (deadline time.Time, ok bool) {}
func (d *dc) Done() <-chan struct{}                   {}
func (d *dc) Err() error                              {}
func (d *dc) Value(key interface{}) interface{}       { return d.Context.Value(key) }

func contextWithTimeout(ctx context.Context, d time.Duration) (context.Context, func()) {
	wrapCtx, wrapCancel := context.WithCancel(ctx)
	d := dc{wrapCtx}
	cancel := func() {
		wrapCancel()
		d.Cancel()
	}
	return d, cancel
}
