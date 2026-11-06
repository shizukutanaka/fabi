package main

import (
	"context"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// Finding is one discovered thing: a tool, service, package, model, or GPU.
type Finding struct {
	Kind      string // tool | service | package | model | gpu | runtime
	Name      string
	Detail    string
	SizeBytes int64
	Path      string
	Running   bool
	Metrics   map[string]float64 // numeric measurements (gpu vram/util, ...)
}

type ProbeResult struct {
	Name     string
	Duration time.Duration
	Findings []Finding
	Err      error
}

type probe struct {
	name string
	fn   func(ctx context.Context) ([]Finding, error)
}

func allProbes() []probe {
	return []probe{
		{"process", probeProcesses},
		{"port", probePorts},
		{"filesystem", probeFilesystem},
		{"package", probePackages},
		{"gpu", probeGPU},
		{"models", probeModels},
	}
}

// runScan executes all probes, writes the ledger, and prints the report.
// quiet prints a one-line summary instead of the full report (watch mode).
func runScan(start time.Time, quiet bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2800*time.Millisecond)
	defer cancel()

	ledger := openLedger()
	defer ledger.close()
	ledger.scanTs = start.UnixMilli()
	ledger.emit("scan.started", map[string]any{
		"fabi_version": version,
		"goos":         runtime.GOOS,
		"goarch":       runtime.GOARCH,
	})

	results := make([]ProbeResult, 0, 6)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, p := range allProbes() {
		wg.Add(1)
		go func(p probe) {
			defer wg.Done()
			t0 := time.Now()
			findings, err := p.fn(ctx)
			res := ProbeResult{Name: p.name, Duration: time.Since(t0), Findings: findings, Err: err}
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
			ledger.emit("probe.completed", map[string]any{
				"probe":        p.name,
				"duration_ms":  res.Duration.Milliseconds(),
				"findings":     len(findings),
				"error":        errString(err),
			})
		}(p)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })

	var all []Finding
	for _, r := range results {
		all = append(all, r.Findings...)
		for _, f := range r.Findings {
			pl := map[string]any{
				"kind":       f.Kind,
				"name":       f.Name,
				"detail":     f.Detail,
				"size_bytes": f.SizeBytes,
				"running":    f.Running,
				"path":       f.Path,
			}
			if f.Metrics != nil {
				pl["metrics"] = f.Metrics
			}
			ledger.emit("resource.found", pl)
		}
	}

	var vm vmStat
	if v, err := getVirtualMemory(); err == nil {
		vm = v
		ledger.emit("measurement.sampled", map[string]any{
			"ram_total_bytes":     vm.Total,
			"ram_used_bytes":      vm.Used,
			"ram_available_bytes": vm.Available,
		})
	}
	var rss uint64
	if r, err := selfRSS(); err == nil {
		rss = r
		ledger.emit("measurement.sampled", map[string]any{
			"fabi_rss_bytes": rss,
		})
	}
	for _, f := range all {
		if f.Kind == "gpu" && f.Metrics != nil {
			ledger.emit("measurement.sampled", map[string]any{
				"gpu":            f.Name,
				"vram_used_mib":  f.Metrics["vram_used_mib"],
				"vram_total_mib": f.Metrics["vram_total_mib"],
				"gpu_util":       f.Metrics["gpu_util"],
				"gpu_temp_c":     f.Metrics["gpu_temp_c"],
			})
		}
	}

	elapsed := time.Since(start)
	ledger.emit("report.generated", map[string]any{
		"findings":    len(all),
		"duration_ms": elapsed.Milliseconds(),
	})
	if quiet {
		printScanLine(all, vm, elapsed)
	} else {
		printReport(buildReport(results, all, elapsed, vm, rss), results, elapsed)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func selfRSS() (uint64, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, err
	}
	mi, err := p.MemoryInfo()
	if err != nil {
		return 0, err
	}
	return mi.RSS, nil
}
