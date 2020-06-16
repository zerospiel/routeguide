package main

import (
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

// BalancerName is a default name for a WRR balancer.
const BalancerName = "ozwrr"

type (
	pickerBldr struct{}
)

func init() {
	balancer.Register(newPBuilder(BalancerName))
}

func newPBuilder(name string) balancer.Builder {
	return base.NewBalancerBuilderV2(name, &pickerBldr{}, base.Config{HealthCheck: true})
}

// Build returns a picker that will be used by gRPC to pick a SubConn.
//
// It's impossible to pass original Target from resolver.Resolver to Picker.
// To support mesh version Picker has to know original service name (ex: catalog-api.bx),
// so the only way to pass it here is to use Address.Attributes.
// PickerBuilder expects that all addresses belongs to the same service.
func (p *pickerBldr) Build(info base.PickerBuildInfo) balancer.V2Picker {
	pool := newConnSet()

	for conn := range info.ReadySCs {
		pool.add(conn)
	}

	return &wrrPicker{
		p: pool,
	}
}

type connSet struct {
	mu   sync.Mutex
	sc   []balancer.SubConn
	next int
}

func newConnSet() *connSet {
	// TODO: set random next depending on readyscs pool size
	return &connSet{
		sc: make([]balancer.SubConn, 0),
	}
}

func (cs *connSet) add(sc balancer.SubConn) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.sc = append(cs.sc, sc)
}

func (cs *connSet) pick() (balancer.PickResult, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if len(cs.sc) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	// default rr
	sc := cs.sc[cs.next]
	cs.next = (cs.next + 1) % len(cs.sc)

	return balancer.PickResult{SubConn: sc}, nil
}

type wrrPicker struct {
	p *connSet
}

// Pick returns the connection to use for this RPC and related information.
func (p *wrrPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	return p.p.pick()
}

func addressAttr(a resolver.Address, attr string) (value string, exists bool) {
	value, exists = a.Attributes.Value(attr).(string)

	return value, exists
}
