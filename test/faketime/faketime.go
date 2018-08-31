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

// Package faketime implements a fake version of grpctime.
package faketime

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/internal/grpctime"
)

func Uninstall() {
	grpctime.AfterFunc = time.AfterFunc
	grpctime.NewTicker = time.NewTicker
	grpctime.NewTimer = time.NewTimer
	grpctime.Now = time.Now
	grpctime.Sleep = time.Sleep
	grpctime.ContextWithTimeout = context.WithTimeout
}

func Install() {
	grpctime.AfterFunc = afterFunc
	grpctime.NewTicker = newTicker
	grpctime.NewTimer = newTimer
	grpctime.Now = now
	grpctime.Sleep = sleep
	grpctime.ContextWithTimeout = contextWithTimeout
}

var mu sync.Mutex
var now time.Time
var updated = sync.NewCond(mu)

func afterFunc(d time.Duration, f func()) *time.Timer {
}

func newTicker(d time.Duration) *time.Ticker {
}

func newTimer(d time.Duration) *grpctime.Timer {
}

func now() time.Time {
	mu.Lock()
	defer mu.Unlock()
	return now
}

func sleep(d time.Duration) {
	start := now
	mu.Lock()
	for now.Before(d) {
		updated.Wait()
	}
	mu.Unlock()
}
