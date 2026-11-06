package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// ---- shared detection data (data, not code — SPEC v0.6 L6) ----

type portTarget struct {
	Port int    `json:"port"`
	Name string `json:"name"`
}

// expandDir resolves "~/" to the home dir and %ENV% vars cross-platform.
func expandDir(d string) string {
	if strings.HasPrefix(d, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, d[2:])
	}
	out := d
	for {
		i := strings.Index(out, "%")
		if i < 0 {
			break
		}
		j := strings.Index(out[i+1:], "%")
		if j < 0 {
			break
		}
		j += i + 1
		out = out[:i] + os.Getenv(out[i+1:j]) + out[j+1:]
	}
	return filepath.Clean(out)
}

var modelExts = map[string]bool{
	".gguf": true, ".safetensors": true, ".exl2": true, ".awq": true, ".gptq": true,
}

// wslDistro is shared by wsl_windows.go (real) and wsl_other.go (stub).
type wslDistro struct {
	name    string
	vhdx    string
	present bool
}

// ---- probe 1: processes ----

// procGroup aggregates same-workload processes: one finding per group.
type procGroup struct {
	name    string
	kind    string
	rss     uint64
	pids    []int32
	pattern string
}

func probeProcesses(ctx context.Context) ([]Finding, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}
	groups := map[string]*procGroup{}
	order := []string{}
	add := func(key, name, kind, pattern string, pid int32, rss uint64) {
		g, ok := groups[key]
		if !ok {
			g = &procGroup{name: name, kind: kind, pattern: pattern}
			groups[key] = g
			order = append(order, key)
		}
		g.rss += rss
		g.pids = append(g.pids, pid)
	}

	ps := loadPatterns()
	for _, p := range procs {
		if ctx.Err() != nil {
			break
		}
		name, err := p.Name()
		if err != nil {
			continue
		}
		lower := strings.ToLower(name)
		var rss uint64
		if mi, err := p.MemoryInfo(); err == nil {
			rss = mi.RSS
		}

		kind := "tool"
		if strings.Contains(lower, "docker") {
			kind = "runtime"
		}
		matched := false
		for _, pat := range ps.ExePatterns {
			if strings.Contains(lower, pat) {
				matched = true
				break
			}
		}
		if matched {
			add("exe:"+lower, name, kind, "", p.Pid, rss)
			continue
		}

		// Interpreter workloads: python.exe running a known AI script.
		if isScriptHost(lower) {
			if cmd, err := p.Cmdline(); err == nil {
				cl := strings.ToLower(cmd)
				for _, pat := range ps.CmdPatterns {
					if strings.Contains(cl, pat) {
						add("wl:"+lower+":"+pat, name+" ["+pat+"]", "tool", pat, p.Pid, rss)
						break
					}
				}
			}
		}
	}

	var out []Finding
	for _, key := range order {
		g := groups[key]
		pids := g.pids
		pidStr := fmt.Sprintf("%d", pids[0])
		if len(pids) > 1 {
			pidStr = fmt.Sprintf("%d+%d more", pids[0], len(pids)-1)
		}
		out = append(out, Finding{
			Kind: g.kind, Name: g.name,
			Detail:  fmt.Sprintf("%d proc(s), %.0f MB total (pid %s)", len(g.pids), float64(g.rss)/1048576, pidStr),
			Running: true,
		})
	}
	return out, nil
}

func isScriptHost(name string) bool {
	base := strings.TrimSuffix(strings.ToLower(name), ".exe")
	for _, h := range loadPatterns().ScriptHosts {
		if base == h {
			return true
		}
	}
	return false
}

// ---- probe 2: localhost ports ----

func probePorts(ctx context.Context) ([]Finding, error) {
	owners := listeningOwners(ctx)
	var out []Finding
	var mu sync.Mutex
	var wg sync.WaitGroup
	invisible := 0
	for _, t := range loadPatterns().Ports {
		wg.Add(1)
		go func(t portTarget) {
			defer wg.Done()
			addr := fmt.Sprintf("127.0.0.1:%d", t.Port)
			conn, err := net.DialTimeout("tcp", addr, 150*time.Millisecond)
			if err != nil {
				return
			}
			conn.Close()
			detail := addr
			if sig := httpIdentify(addr); sig != "" {
				detail = fmt.Sprintf("%s — %s", addr, sig)
			}
			if pid, ok := owners[t.Port]; ok {
				if name, err := procNameByPID(pid); err == nil {
					detail = fmt.Sprintf("%s — owner %s (pid %d)", detail, name, pid)
				} else {
					detail = fmt.Sprintf("%s — owner pid %d", detail, pid)
				}
			} else {
				detail = fmt.Sprintf("%s — owner not visible (VM/WSL/container?)", detail)
				mu.Lock()
				invisible++
				mu.Unlock()
			}
			mu.Lock()
			out = append(out, Finding{
				Kind: "service", Name: t.Name,
				Detail: detail, Running: true,
			})
			mu.Unlock()
		}(t)
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if invisible > 0 {
		out = append(out, Finding{
			Kind: "runtime", Name: "hidden compute (inferred)",
			Detail: fmt.Sprintf("%d localhost AI service(s) answered with no visible owner process — likely WSL2, VM, or container", invisible),
		})
	}
	return out, nil
}

// httpIdentify asks a service who it is: GET /, capture status + body head.
func httpIdentify(addr string) string {
	status, b, err := rawHTTP(addr, "GET", "/", "", 350*time.Millisecond)
	if err != nil || status == 0 {
		return ""
	}
	snippet := strings.TrimSpace(string(b))
	if len(snippet) > 80 {
		snippet = snippet[:80] + "…"
	}
	return fmt.Sprintf("HTTP %d | %s", status, snippet)
}

// listeningOwners maps localhost port → PID via netstat (one spawn, ~50 ms).
func listeningOwners(ctx context.Context) map[int]int32 {
	m := map[int]int32{}
	cmd := exec.CommandContext(ctx, "netstat", "-ano")
	b, err := cmd.Output()
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 5 || f[0] != "TCP" {
			continue
		}
		if !strings.EqualFold(f[3], "LISTENING") {
			continue
		}
		// local address is f[1], form "127.0.0.1:1234" or "[::1]:1234"
		local := f[1]
		i := strings.LastIndex(local, ":")
		if i < 0 {
			continue
		}
		var port int
		if _, err := fmt.Sscanf(local[i+1:], "%d", &port); err != nil {
			continue
		}
		var pid int32
		if _, err := fmt.Sscanf(f[4], "%d", &pid); err == nil && pid > 0 {
			m[port] = pid
		}
	}
	return m
}

func procNameByPID(pid int32) (string, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return "", err
	}
	return p.Name()
}

// ---- probe 3: filesystem presence ----

func probeFilesystem(ctx context.Context) ([]Finding, error) {
	var out []Finding
	for _, raw := range loadPatterns().ScanDirs {
		if ctx.Err() != nil {
			break
		}
		d := expandDir(raw)
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			out = append(out, Finding{
				Kind: "tool", Name: dirDisplayName(d),
				Detail: d, Running: false,
			})
		}
	}
	// WSL2 distros: registered name + vhdx presence — dead images are findings.
	for _, d := range wslDistros(ctx) {
		if d.present {
			out = append(out, Finding{
				Kind: "runtime", Name: "WSL distro: " + d.name,
				Detail: d.vhdx,
			})
		} else {
			out = append(out, Finding{
				Kind: "runtime", Name: "WSL distro: " + d.name,
				Detail: "vhdx MISSING — registered but disk image gone (dead install)",
				Path:   d.vhdx,
			})
		}
	}
	return out, nil
}

func dirDisplayName(path string) string {
	base := strings.ToLower(filepath.Base(path))
	parent := strings.ToLower(filepath.Base(filepath.Dir(path)))
	switch {
	case strings.Contains(base, "ollama"):
		return "Ollama"
	case strings.Contains(base, "lm-studio"), strings.Contains(base, "lmstudio"),
		strings.Contains(base, "lm studio"):
		return "LM Studio"
	case strings.Contains(base, "comfyui"):
		return "ComfyUI"
	case strings.Contains(base, "gpt4all"):
		return "GPT4All"
	case strings.Contains(base, "anythingllm"):
		return "AnythingLLM"
	case base == "jan" || strings.Contains(base, "jan.ai"):
		return "Jan"
	case strings.Contains(base, "dify"):
		return "Dify"
	case strings.Contains(base, "flowise"):
		return "Flowise"
	case strings.Contains(base, "pinokio"):
		return "Pinokio"
	case strings.Contains(base, "huggingface") || strings.Contains(parent, "huggingface"):
		return "HuggingFace cache"
	default:
		return filepath.Base(path)
	}
}

// ---- probe 4: installed packages ----

func probePackages(ctx context.Context) ([]Finding, error) {
	pkgs, err := installedPackages(ctx)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, p := range pkgs {
		if ctx.Err() != nil {
			break
		}
		lower := strings.ToLower(p.name)
		kind := ""
		for _, pat := range loadPatterns().PkgAITools {
			if strings.Contains(lower, pat) {
				kind = "tool"
				break
			}
		}
		if kind == "" {
			for _, pat := range loadPatterns().PkgRuntimes {
				if strings.Contains(lower, pat) {
					kind = "runtime"
					break
				}
			}
		}
		if kind == "" {
			continue
		}
		out = append(out, Finding{
			Kind:   kind,
			Name:   p.name,
			Detail: fmt.Sprintf("%s %s — %s", p.name, p.version, p.source),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

type pkgInfo struct {
	name    string
	version string
	source  string
}

// ---- probe 5: GPU ----

func probeGPU(ctx context.Context) ([]Finding, error) {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil, nil // no NVIDIA toolchain visible; other vendors later
	}
	// nvidia-smi can stall in driver ioctls under heavy GPU load — cap at
	// 1500 ms so a busy driver can never stall the whole scan.
	gctx, gcancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer gcancel()
	cmd := exec.CommandContext(gctx, "nvidia-smi",
		"--query-gpu=name,memory.total,memory.used,utilization.gpu,temperature.gpu",
		"--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi: %w", err)
	}
	var findings []Finding
	findings = append(findings, parseGPUHeader(out)...)
	// Per-process GPU attribution: VRAM burn is a physics detection channel —
	// an AI tool that matches no name pattern is still caught by usage.
	var apps []Finding
	if a, err := gpuComputeApps(gctx); err == nil {
		apps = a
	}
	// WDDM attribution gap: header reports VRAM used but per-process list
	// is empty. Missing provenance → inference note, never silence (L2).
	var appSum float64
	for _, a := range apps {
		appSum += a.Metrics["vram_used_mib"]
	}
	for _, f := range findings {
		if f.Kind == "gpu" && f.Metrics != nil {
			if gap := f.Metrics["vram_used_mib"] - appSum; gap > 1024 {
				findings = append(findings, Finding{
					Kind:   "gpu",
					Name:   f.Name + " — unattributed use",
					Detail: fmt.Sprintf("%.0f MiB VRAM in use by unattributed processes (WDDM/VM attribution gap)", gap),
				})
			}
		}
	}
	findings = append(findings, apps...)
	return findings, nil
}

func parseGPUHeader(out []byte) []Finding {
	var findings []Finding
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Split(line, ",")
		if len(fields) < 5 {
			continue
		}
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		var vramTotal, vramUsed, util, temp float64
		fmt.Sscanf(fields[1], "%f", &vramTotal)
		fmt.Sscanf(fields[2], "%f", &vramUsed)
		fmt.Sscanf(fields[3], "%f", &util)
		fmt.Sscanf(fields[4], "%f", &temp)
		m := map[string]float64{
			"vram_total_mib": vramTotal,
			"vram_used_mib":  vramUsed,
			"gpu_util":       util,
			"gpu_temp_c":     temp,
		}
		findings = append(findings, Finding{
			Kind: "gpu", Name: fields[0],
			Detail: fmt.Sprintf("VRAM %s/%s MiB, util %s%%, %s°C",
				fields[2], fields[1], fields[3], fields[4]),
			Metrics: m,
		})
	}
	return findings
}

// gpuComputeApps returns findings for every process burning VRAM.
func gpuComputeApps(ctx context.Context) ([]Finding, error) {
	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-compute-apps=pid,process_name,used_memory",
		"--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			continue
		}
		pidStr := strings.TrimSpace(fields[0])
		pname := strings.TrimSpace(fields[1])
		var pid int32
		fmt.Sscanf(pidStr, "%d", &pid)
		var vram float64
		fmt.Sscanf(strings.TrimSpace(fields[2]), "%f", &vram)
		// WDDM reports every graphics client at ~0 MiB — only real VRAM
		// consumption counts as a compute workload.
		if vram <= 0 {
			continue
		}
		// Windows nvidia-smi returns the exe path — take the basename.
		name := filepath.Base(pname)
		findings = append(findings, Finding{
			Kind: "tool", Name: name + " [gpu-client]",
			Detail:  fmt.Sprintf("pid %d, %.0f MiB VRAM in use", pid, vram),
			Running: true,
			Metrics: map[string]float64{"vram_used_mib": vram},
		})
	}
	return findings, nil
}

// ---- probe 6: models ----

func probeModels(ctx context.Context) ([]Finding, error) {
	var out []Finding

	// Ollama model manifests: path encodes registry/model/tag.
	if home, err := os.UserHomeDir(); err == nil {
		manifests := filepath.Join(home, ".ollama", "models", "manifests")
		if st, err := os.Stat(manifests); err == nil && st.IsDir() {
			filepath.WalkDir(manifests, func(path string, d fs.DirEntry, err error) error {
				if err != nil || ctx.Err() != nil || d.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(manifests, path)
				name := strings.ReplaceAll(rel, string(filepath.Separator), "/")
				name = strings.TrimPrefix(name, "registry.ollama.ai/")
				if i := strings.LastIndex(name, "/"); i >= 0 {
					name = name[:i] + ":" + name[i+1:]
				}
				out = append(out, Finding{Kind: "model", Name: name, Detail: "ollama", Path: path})
				return nil
			})
			// Total blob store size (bounded).
			blobs := filepath.Join(home, ".ollama", "models", "blobs")
			if size, n, _ := boundedSize(blobs, ctx); size > 0 {
				out = append(out, Finding{
					Kind: "model", Name: "ollama blob store",
					Detail: fmt.Sprintf("%d files", n), SizeBytes: size, Path: blobs,
				})
			}
		}
	}

	// Generic model-file roots (LM Studio, GPT4All, HF cache, ComfyUI, …).
	var roots []string
	for _, raw := range loadPatterns().ModelRoots {
		roots = append(roots, expandDir(raw))
	}
	// ComfyUI keeps models inside its own dir — add <comfyui>\models for every
	// ComfyUI install the filesystem probe can see.
	for _, raw := range loadPatterns().ScanDirs {
		d := expandDir(raw)
		if strings.Contains(strings.ToLower(d), "comfyui") {
			roots = append(roots, filepath.Join(d, "models"))
		}
	}
	for _, root := range roots {
		if ctx.Err() != nil {
			break
		}
		if st, err := os.Stat(root); err != nil || !st.IsDir() {
			continue
		}
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || ctx.Err() != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if !modelExts[ext] {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			out = append(out, Finding{
				Kind: "model", Name: d.Name(),
				Detail: filepath.Base(root), SizeBytes: info.Size(), Path: path,
			})
			return nil
		})
	}
	return out, nil
}

// boundedSize sums file sizes under root until ctx ends.
// Returns bytes, file count, and whether the walk completed.
func boundedSize(root string, ctx context.Context) (int64, int, bool) {
	var total int64
	var count int
	complete := true
	if _, err := os.Stat(root); err != nil {
		return 0, 0, true
	}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			complete = false
			return fs.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
			count++
		}
		return nil
	})
	return total, count, complete
}

// ---- small helpers used by report ----

// decodeManifest is reserved for future manifest parsing.
var _ = json.Marshal
