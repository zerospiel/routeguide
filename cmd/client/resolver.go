package main

import (
	"fmt"
	"strings"

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
		for _, a := range strings.Split(serverIPs, ",") {
			// WARN: deprecated usage but only for fast build purposes
			addresses = append(addresses, resolver.Address{Addr: a, Type: resolver.Backend})
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
