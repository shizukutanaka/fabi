package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeToolName(t *testing.T) {
	cases := map[string]string{
		"ollama.exe":               "ollama",
		".ollama":                  "ollama",
		"Ollama":                   "ollama",
		"LM Studio":                "lm studio",
		"lm-studio.exe":            "lm studio",
		"python.exe [strata]":      "python|strata",
		"python.exe (ai-workload)": "python",
		"ComfyUI":                  "comfyui",
	}
	for in, want := range cases {
		if got := normalizeToolName(in); got != want {
			t.Errorf("normalizeToolName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeRuntimeKey(t *testing.T) {
	cases := map[string]string{
		"Python 3.11.9 (64-bit)":         "python@3.11",
		"Python 3.11.9 Core Interpreter": "python@3.11",
		"Python 3.14.7 (32-bit)":         "python@3.14",
		"Git 2.56.0.4":                   "git@2.56",
		"Visual Studio Code 1.96":        "visual@1.96",
		"Python Launcher":                "python",
	}
	for in, want := range cases {
		if got := normalizeRuntimeKey(in); got != want {
			t.Errorf("normalizeRuntimeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpandDir(t *testing.T) {
	if got := expandDir("~/x"); !strings.HasSuffix(got, "x") {
		t.Errorf("expandDir(~/x) = %q", got)
	}
	if runtime.GOOS == "windows" {
		if got := expandDir("%APPDATA%/x"); !strings.Contains(strings.ToLower(got), "appdata") {
			t.Errorf("expandDir(%%APPDATA%%/x) = %q", got)
		}
	}
}

func TestEvalTask(t *testing.T) {
	tp := taskProfiles["reasoning"]
	r := evalTask(tp, "NVIDIA RTX A2000 12GB", 12000)
	if r.toks <= 0 {
		t.Fatalf("toks should be positive, got %f", r.toks)
	}
	// 288 GB/s ÷ 4.7 GB ≈ 61 tok/s
	if r.toks < 50 || r.toks > 75 {
		t.Errorf("toks %f out of expected range", r.toks)
	}
	if r.verdict != "TOO SLOW" {
		t.Errorf("reasoning on A2000 should be TOO SLOW, got %q", r.verdict)
	}
}

func TestBandwidthForGPU(t *testing.T) {
	if bw := bandwidthForGPU("NVIDIA RTX A2000 12GB"); bw != 288 {
		t.Errorf("A2000 bandwidth = %f, want 288", bw)
	}
	if bw := bandwidthForGPU("GeForce RTX 4090"); bw != 1008 {
		t.Errorf("4090 bandwidth = %f, want 1008", bw)
	}
	if bw := bandwidthForGPU("Unknown GPU"); bw != 200 {
		t.Errorf("unknown bandwidth = %f, want 200 default", bw)
	}
}

// whyAction — the command that closes the observe→recommend→act loop.
// Every class must map to a concrete command, not an empty string.
func TestWhyAction(t *testing.T) {
	cases := map[string]string{
		"idle":        "fabi unload ollama",
		"remove":      "fabi report (then manual removal — L9)",
		"investigate": "fabi scan",
		"info":        "fabi auto",
		"unknown":     "fabi auto", // falls through to the safe default
	}
	for class, want := range cases {
		if got := whyAction(class, "ollama"); got != want {
			t.Errorf("whyAction(%q) = %q, want %q", class, got, want)
		}
	}
}

// computeRecommendations — the 8 evidence rules over scan history.
// No findings → no recommendations; dup models → a remove rec.
func TestComputeRecommendations(t *testing.T) {
	// Empty estate produces no recommendations (no evidence).
	windows := []scanWindow{{findings: nil}}
	if recs := computeRecommendations(nil, windows, windows[0]); len(recs) != 0 {
		t.Errorf("empty estate produced %d recs, want 0", len(recs))
	}
	// Two identical-name models in the latest scan → a remove rec.
	findings := []findingRec{
		{kind: "model", name: "qwen2", size: 100, path: "/a"},
		{kind: "model", name: "qwen2", size: 100, path: "/b"},
	}
	w := []scanWindow{{findings: findings}}
	recs := computeRecommendations(nil, w, w[0])
	var found bool
	for _, r := range recs {
		if r.class == "remove" || r.class == "investigate" {
			found = true
		}
	}
	if !found {
		t.Errorf("dup models produced no rec, got %v", recs)
	}
}
