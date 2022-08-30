/*
 *
 * Copyright 2020 gRPC authors.
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

package testutils

import (
	"context"
	"reflect"
	"testing"
)

// DefaultChanBufferSize is the default buffer size of the underlying channel.
const DefaultChanBufferSize = 1

// Channel wraps a generic channel and provides a timed receive operation.
type Channel struct {
	ch reflect.Value
}

// Send sends value on the underlying channel.
func (c *Channel) Send(value interface{}) {
	c.ch.Send(reflect.ValueOf(&value).Elem())
}

// SendContext sends value on the underlying channel, or returns an error if
// the context expires.
func (c *Channel) SendContext(ctx context.Context, value interface{}) error {
	chosen, _, _ := reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectSend, Chan: c.ch, Send: reflect.ValueOf(&value).Elem()},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
	})
	if chosen == 1 {
		return ctx.Err()
	}
	return nil
}

// SendOrFail attempts to send value on the underlying channel.  Returns true
// if successful or false if the channel was full.
func (c *Channel) SendOrFail(value interface{}) bool {
	chosen, _, _ := reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectSend, Chan: c.ch, Send: reflect.ValueOf(&value).Elem()},
		{Dir: reflect.SelectDefault},
	})
	return chosen == 0
}

// ReceiveOrFail returns the value on the underlying channel and true, or nil
// and false if the channel was empty.
func (c *Channel) ReceiveOrFail() (interface{}, bool) {
	chosen, recv, _ := reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: c.ch},
		{Dir: reflect.SelectDefault},
	})
	if chosen == 1 {
		return nil, false
	}
	return recv.Interface(), true
}

// ReceiveOrFatal returns the value on the underlying channel if it is received
// before the context expires, or calls t.Fatal.
func (c *Channel) ReceiveOrFatal(ctx context.Context, t *testing.T) interface{} {
	t.Helper()
	val, err := c.Receive(ctx)
	if err != nil {
		t.Fatalf("Context canceled waiting on channel receive: %v", err)
	}
	return val
}

// Receive returns the value received on the underlying channel, or the error
// returned by ctx if it is closed or cancelled.
func (c *Channel) Receive(ctx context.Context) (interface{}, error) {
	chosen, recv, _ := reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: c.ch},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
	})
	if chosen == 1 {
		return nil, ctx.Err()
	}
	return recv.Interface(), nil
}

// Replace clears the value on the underlying channel, and sends the new value.
//
// It's expected to be used with a size-1 channel, to only keep the most
// up-to-date item. This method is inherently racy when invoked concurrently
// from multiple goroutines.
func (c *Channel) Replace(value interface{}) {
	v := reflect.ValueOf(&value).Elem()
	for {
		chosen, _, _ := reflect.Select([]reflect.SelectCase{
			{Dir: reflect.SelectSend, Chan: c.ch, Send: v},
			{Dir: reflect.SelectRecv, Chan: c.ch},
		})
		if chosen == 0 {
			return
		}
	}
}

// NewChannel returns a new Channel.
func NewChannel() *Channel {
	return NewChannelWithSize(DefaultChanBufferSize)
}

// NewChannelWithSize returns a new Channel with a buffer of bufSize.
func NewChannelWithSize(bufSize int) *Channel {
	return &Channel{ch: reflect.ValueOf(make(chan interface{}, bufSize))}
}

// ChannelFrom creates a Channel wrapping an existing channel (which must be a
// channel of any type).
func ChannelFrom(channel interface{}) *Channel {
	return &Channel{ch: reflect.ValueOf(channel)}
}

func ReceiveOrFatal(ctx context.Context, t *testing.T, channel interface{}) interface{} {
	t.Helper()
	return ChannelFrom(channel).ReceiveOrFatal(ctx, t)
}
