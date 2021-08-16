package types_test

import (
	"testing"

	"ddosify.com/hammer/core/types"
)

func newDummyHammer() types.Hammer {
	return types.Hammer{
		Proxy:             types.Proxy{Strategy: "single"},
		Packet:            types.Packet{Protocol: "HTTP", Method: "GET"},
		ReportDestination: "stdout",
	}
}

func TestHammerValidAttackType(t *testing.T) {
	var loadTypes = [...]string{"linear", "capacity", "stress", "soak"}

	for _, l := range loadTypes {
		h := newDummyHammer()
		h.LoadType = l

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidAttackType errored: %v", err)
		}
	}
}

func TestHammerInValidAttackType(t *testing.T) {
	h := newDummyHammer()
	h.LoadType = "incremental"

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidAttackType should errored")
	}
}

func TestHammerValidReportDestination(t *testing.T) {
	var supportedDestinations = [...]string{"stdout", "timescale"}

	for _, rd := range supportedDestinations {
		h := newDummyHammer()
		h.ReportDestination = rd

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidReportDestination errored: %v", err)
		}
	}
}

func TestHammerInValidReportDestination(t *testing.T) {
	h := newDummyHammer()
	h.ReportDestination = "output_dummy"

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidReportDestination should errored")
	}
}

func TestHammerValidProxy(t *testing.T) {
	var availableProxyStrategies = [...]string{"single"}

	for _, ps := range availableProxyStrategies {
		h := newDummyHammer()
		h.Proxy = types.Proxy{Strategy: ps}

		if err := h.Validate(); err != nil {
			t.Errorf("TestHammerValidProxy errored: %v", err)
		}
	}
}

func TestHammerInValidProxy(t *testing.T) {
	h := newDummyHammer()
	h.Proxy = types.Proxy{Strategy: "dummy_strategy"}

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidProxy should errored")
	}
}

func TestHammerValidPacket(t *testing.T) {
	var supportedProtocols = [...]string{"HTTP", "HTTPS"}
	var supportedProtocolMethods = map[string][]string{
		"HTTP":  {"GET", "POST", "PUT", "DELETE", "UPDATE", "PATCH"},
		"HTTPS": {"GET", "POST", "PUT", "DELETE", "UPDATE", "PATCH"}}

	for _, p := range supportedProtocols {
		for _, m := range supportedProtocolMethods[p] {
			h := newDummyHammer()
			h.Packet = types.Packet{Protocol: p, Method: m}

			if err := h.Validate(); err != nil {
				t.Errorf("TestHammerValidProtocol errored: %v", err)
			}
		}
	}
}

func TestHammerInValidPacket(t *testing.T) {
	// Incorrect Protocol
	h := newDummyHammer()
	h.Packet = types.Packet{Protocol: "dummy_protocol", Method: "GET"}

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidPacket incorrect protocol should errored")
	}

	// Incorrect Method
	h = newDummyHammer()
	h.Packet = types.Packet{Protocol: "HTTP", Method: "dummy_method"}

	if err := h.Validate(); err == nil {
		t.Errorf("TestHammerInValidPacket incorrect method should errored")
	}
}
