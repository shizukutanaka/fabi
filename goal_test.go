package main

import (
	"math"
	"testing"
)

// TestEvalTaskVerdicts covers the verdict branches main_test.go does
// not: LOCAL OK, POSSIBLE (slow), DOESN'T FIT, and the image-gen path
// whose verdict depends on VRAM fit alone.
func TestEvalTaskVerdicts(t *testing.T) {
	tests := []struct {
		name     string
		tp       taskProfile
		gpu      string
		vramMiB  float64
		wantFit  bool
		verdict  string
		wantToks float64 // checked only when > 0, with 1% tolerance
	}{
		{
			name: "ocr on a 4090 is comfortably local",
			tp:   taskProfiles["ocr"], gpu: "NVIDIA GeForce RTX 4090", vramMiB: 24576,
			wantFit: true, verdict: "LOCAL OK",
			wantToks: 1008e9 / float64(modelBytes["3b"]),
		},
		{
			name: "summarize on a 3060 fits but misses the latency target",
			tp:   taskProfiles["summarize"], gpu: "GeForce RTX 3060", vramMiB: 12288,
			wantFit: true, verdict: "POSSIBLE (slow)",
		},
		{
			name: "70b video model does not fit in 24 GiB",
			tp:   taskProfiles["video"], gpu: "NVIDIA GeForce RTX 4090", vramMiB: 24576,
			wantFit: false, verdict: "DOESN'T FIT",
		},
		{
			name: "image gen verdict is VRAM fit only",
			tp:   taskProfiles["image_gen"], gpu: "anything", vramMiB: 8192,
			wantFit: true, verdict: "VRAM FIT (gen time varies)",
		},
		{
			name: "image gen that exceeds VRAM does not fit",
			tp:   taskProfiles["image_gen"], gpu: "anything",
			vramMiB: float64(modelBytes["img"])/1048576 - 1,
			wantFit: false, verdict: "DOESN'T FIT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalTask(tt.tp, tt.gpu, tt.vramMiB)
			if got.fitVRAM != tt.wantFit {
				t.Errorf("fitVRAM = %v, want %v", got.fitVRAM, tt.wantFit)
			}
			if got.verdict != tt.verdict {
				t.Errorf("verdict = %q, want %q", got.verdict, tt.verdict)
			}
			if tt.wantToks > 0 && math.Abs(got.toks-tt.wantToks)/tt.wantToks > 0.01 {
				t.Errorf("toks = %v, want ~%v", got.toks, tt.wantToks)
			}
			if tt.tp.modelClass != "img" && got.fitVRAM {
				wantLatency := float64(tt.tp.outTokens) / got.toks
				if math.Abs(got.latency-wantLatency) > 1e-9 {
					t.Errorf("latency = %v, want %v (outTokens/toks)", got.latency, wantLatency)
				}
			}
		})
	}
}

func TestEvalTaskVRAMBoundary(t *testing.T) {
	// A model fits when its resident bytes are exactly the VRAM budget.
	exact := float64(modelBytes["8b"]) / 1048576
	if got := evalTask(taskProfiles["summarize"], "GeForce RTX 3060", exact); !got.fitVRAM {
		t.Errorf("resident == VRAM budget should fit, got verdict %q", got.verdict)
	}
	if got := evalTask(taskProfiles["summarize"], "GeForce RTX 3060", exact-1); got.fitVRAM {
		t.Error("resident one MiB over the VRAM budget should not fit")
	}
}
