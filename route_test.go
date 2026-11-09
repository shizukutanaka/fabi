package main

import "testing"

// TestBandwidthForGPUPriority covers the table branches main_test.go
// does not: the remaining SKUs, and the ordering-sensitive cases where
// a specific entry must win over the generic "rtx" fallback.
func TestBandwidthForGPUPriority(t *testing.T) {
	cases := map[string]float64{
		"NVIDIA GeForce RTX 3090": 936,
		"NVIDIA A100-SXM4-80GB":   1555,
		"NVIDIA H100 80GB HBM3":   3350,
		// 3060/3070 share an entry that must win over generic "rtx".
		"GeForce RTX 3060 Ti": 448,
		"GeForce RTX 3070":    448,
		// An RTX card with no specific entry gets the conservative default.
		"GeForce RTX 4070 SUPER": 360,
		// No name at all still yields the floor value.
		"": 200,
	}
	for in, want := range cases {
		if got := bandwidthForGPU(in); got != want {
			t.Errorf("bandwidthForGPU(%q) = %v, want %v", in, got, want)
		}
	}
}
