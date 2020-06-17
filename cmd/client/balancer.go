package main

import (
	"sync"

	"github.com/zerospiel/wrrimpl"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
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

	for conn, connInfo := range info.ReadySCs {
		var w int
		if connInfo.Address.Attributes != nil {
			w = connInfo.Address.Attributes.Value("weight").(int)
		}
		pool.add(conn, int64(w))
	}

	return &wrrPicker{
		p: pool,
	}
}

type connSet struct {
	mu  sync.Mutex
	wrr wrrimpl.WRR
}

func newConnSet() *connSet {
	return &connSet{
		wrr: wrrimpl.NewEDF(),
	}
}

func (cs *connSet) add(sc balancer.SubConn, w int64) {
	if w == 0 {
		return
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.wrr.Add(sc, int64(w))
}

func (cs *connSet) pick() (balancer.PickResult, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// wrr on edf
	sc, ok := cs.wrr.Next().(balancer.SubConn)
	if !ok || sc == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	return balancer.PickResult{SubConn: sc}, nil
}

type wrrPicker struct {
	p *connSet
}

// Pick returns the connection to use for this RPC and related information.
func (p *wrrPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	return p.p.pick()
}
