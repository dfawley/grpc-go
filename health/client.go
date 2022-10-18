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

package health

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/internal"
	"google.golang.org/grpc/internal/backoff"
	"google.golang.org/grpc/internal/grpcsync"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
	"google.golang.org/grpc/status"
)

// WithHealthChecking wraps a balancer implementation and intercepts NewSubConn
// calls, installing health checking on those SubConns and controlling the
// state of the SubConns.
func WithHealthChecking(b balancer.Builder) balancer.Builder {
	if _, ok := b.(balancer.ConfigParser); ok {
		return &bb{Builder: b} // TODO: does this need to be a different type??
	}
	return &bb{Builder: b}
}

type bb struct {
	balancer.Builder
}

func (bb *bb) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	wcc := &wrappedCC{ClientConn: cc}
	return &wrappedBalancer{Balancer: bb.Builder.Build(wcc, opts), wcc: wcc}
}

func (bb *bb) ParseConfig(cfg json.RawMessage) (serviceconfig.LoadBalancingConfig, error) {
	if cp, ok := bb.Builder.(ConfigParser); ok {
		return cp(cfg)
	}
	return nil, nil // Unsupported?? TODO!
}

type wrappedBalancer struct {
	Balancer balancer.Balancer
	wcc      *wrappedCC
}

func (w *wrappedBalancer) UpdateSubConnState(sc balancer.SubConn, state balancer.SubConnState) {
	w.wcc.producers[sc].doSomething(state) // that calls w.Balancer.UpdateSubConnState()
}

type wrappedCC struct {
	balancer.ClientConn
	subConnProducers map[balancer.SubConn]*producer
	subConnClosers   map[balancer.SubConn]func()
}

func (w *wrappedCC) NewSubConn(addrs []resolver.Address, opts balancer.NewSubConnOptions) (balancer.SubConn, error) {
	sc, err := w.ClientConn.NewSubConn(addrs, opts)
	if err != nil {
		return nil, err
	}
	pr, close := sc.GetOrBuildProducer(producerBuilderSingleton)
	p := pr.(*producer)
	w.subConnClosers[sc] = close
	w.subConnProducers[sc] = p
	pr.doSomething()
}

func (w *wrappedCC) RemoveSubConn(sc balancer.SubConn) {
	if closer := w.subConnClosers[sc]; closer != nil {
		closer()
		delete(w.subConnClosers, sc)
		delete(w.subConnProducers, sc)
	}
	w.ClientConn.RemoveSubConn(sc)
}

var (
	backoffStrategy = backoff.DefaultExponential
	backoffFunc     = func(ctx context.Context, retries int) bool {
		d := backoffStrategy.Backoff(retries)
		timer := time.NewTimer(d)
		select {
		case <-timer.C:
			return true
		case <-ctx.Done():
			timer.Stop()
			return false
		}
	}
)

func init() {
	balancer.Register(bb{})
	internal.SetSomething()
}

type producerBuilder struct{}

// Build constructs and returns a producer and its cleanup function
func (*producerBuilder) Build(cci interface{}) (balancer.Producer, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &producer{
		client:    healthgrpc.NewOpenRcaServiceClient(cci.(grpc.ClientConnInterface)),
		closed:    grpcsync.NewEvent(),
		intervals: make(map[time.Duration]int),
		listeners: make(map[OOBListener]struct{}),
		backoff:   internal.DefaultBackoffFunc,
	}
	go p.run(ctx)
	return p, func() {
		cancel()
		<-p.closed.Done() // Block until stream stopped.
	}
}

var producerBuilderSingleton = &producerBuilder{}

type producer struct {
}

type producer struct {
	client healthgrpc.HealthClient

	closed *grpcsync.Event // fired when closure completes
	// backoff is called between stream attempts to determine how long to delay
	// to avoid overloading a server experiencing problems.  The attempt count
	// is incremented when stream errors occur and is reset when the stream
	// reports a result.
	backoff func(int) time.Duration

	mu        sync.Mutex
	intervals map[time.Duration]int    // map from interval time to count of listeners requesting that time
	listeners map[OOBListener]struct{} // set of registered listeners
}

const healthCheckMethod = "/grpc.health.v1.Health/Watch"

// This function implements the protocol defined at:
// https://github.com/grpc/grpc/blob/master/doc/health-checking.md
func clientHealthCheck(ctx context.Context, newStream func(string) (interface{}, error), setConnectivityState func(connectivity.State, error), service string) error {
	tryCnt := 0

retryConnection:
	for {
		// Backs off if the connection has failed in some way without receiving a message in the previous retry.
		if tryCnt > 0 && !backoffFunc(ctx, tryCnt-1) {
			return nil
		}
		tryCnt++

		if ctx.Err() != nil {
			return nil
		}
		setConnectivityState(connectivity.Connecting, nil)
		rawS, err := newStream(healthCheckMethod)
		if err != nil {
			continue retryConnection
		}

		s, ok := rawS.(grpc.ClientStream)
		// Ideally, this should never happen. But if it happens, the server is marked as healthy for LBing purposes.
		if !ok {
			setConnectivityState(connectivity.Ready, nil)
			return fmt.Errorf("newStream returned %v (type %T); want grpc.ClientStream", rawS, rawS)
		}

		if err = s.SendMsg(&healthpb.HealthCheckRequest{Service: service}); err != nil && err != io.EOF {
			// Stream should have been closed, so we can safely continue to create a new stream.
			continue retryConnection
		}
		s.CloseSend()

		resp := new(healthpb.HealthCheckResponse)
		for {
			err = s.RecvMsg(resp)

			// Reports healthy for the LBing purposes if health check is not implemented in the server.
			if status.Code(err) == codes.Unimplemented {
				setConnectivityState(connectivity.Ready, nil)
				return err
			}

			// Reports unhealthy if server's Watch method gives an error other than UNIMPLEMENTED.
			if err != nil {
				setConnectivityState(connectivity.TransientFailure, fmt.Errorf("connection active but received health check RPC error: %v", err))
				continue retryConnection
			}

			// As a message has been received, removes the need for backoff for the next retry by resetting the try count.
			tryCnt = 0
			if resp.Status == healthpb.HealthCheckResponse_SERVING {
				setConnectivityState(connectivity.Ready, nil)
			} else {
				setConnectivityState(connectivity.TransientFailure, fmt.Errorf("connection active but health check failed. status=%s", resp.Status))
			}
		}
	}
}
