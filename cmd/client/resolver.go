package main

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/dns"
	"google.golang.org/grpc/resolver/manual"
)

//go:generate stringer -type=resolverType
type resolverType int

const (
	resolverDNS resolverType = iota
	resolverManual

	resolverDNSStr     = "dns"
	resolverManualStr  = "manual"
	resolverUnknownStr = "unknown"
)

func ParseResolverType(rt string) (resolverType, error) {
	switch strings.ToLower(rt) {
	case resolverDNSStr:
		return resolverDNS, nil
	case resolverManualStr:
		return resolverManual, nil
	}

	return -1, fmt.Errorf("Unsupported resolver type: %s", rt)
}

func registerResolver(rt resolverType, serverIPs string) error {
	var builder resolver.Builder
	switch rt {
	case resolverDNS:
		builder = dns.NewBuilder()
	case resolverManual:
		b, _ := manual.GenerateAndRegisterManualResolver()
		addresses := []resolver.Address{}
		for i, a := range strings.Split(serverIPs, ",") {
			ad := resolver.Address{
				Addr:       a,
				Attributes: attributes.New(append(make([]interface{}, 0), "weight", (i+1)*10)...),
				// WARN: deprecated usage but only for fast build purposes
				Type: resolver.Backend,
			}
			if i == 1 {
				ad.Attributes = nil
			}
			addresses = append(addresses, ad)
		}
		b.InitialState(resolver.State{
			Addresses: addresses,
		})
		builder = b
	default:
		return fmt.Errorf("Unsupported resolver type: %s", rt)
	}

	resolver.Register(builder)
	resolver.SetDefaultScheme(builder.Scheme())

	return nil
}
