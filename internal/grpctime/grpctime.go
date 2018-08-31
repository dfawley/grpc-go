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

package grpctime

import (
	"context"
	"time"
)

type Timer interface {
	Reset(d Duration) bool
	Stop() bool
	C() <-chan Time
}

type timer struct {
	time.Timer
}

func (t *timer) C() <-chan Time {
	return t.Timer.C
}

type Ticker interface {
	Stop() bool
	C() <-chan Time
}

type ticker struct {
	time.Ticker
}

func (t *ticker) C() <-chan Time {
	return t.Ticker.C
}

// Wrapped functions from the time/context packages.
var (
	Now   = time.Now
	Sleep = time.Sleep

	AfterFunc = func(d duration, f func()) {
		return &timer{time.AfterFunc(d, f)}
	}
	NewTimer = func(d duration) time.Timer {
		return &timer{time.NewTimer(d)}
	}
	NewTicker = func(d duration) time.Ticker {
		return &ticker{time.NewTicker(d)}
	}

	ContextWithTimeout = context.WithTimeout
)

// After behaves like time.After
func After(d time.Duration) <-chan time.Time {
	return NewTimer(d).C
}

// Since behaves like time.Since
func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}
