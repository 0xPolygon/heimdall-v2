// Package memwatch implements a self-triggering profile capture for the
// pruning-RSS-spike investigation.
//
// When HEIMDALL_PRUNE_AUTOSNAP=1 is set, a goroutine polls /proc/self/status
// every poll-interval. Once VmRSS crosses an absolute or multiplicative
// threshold, it dumps:
//
//   - heap pprof              -> <dir>/heap-<unixnano>.pprof
//   - goroutine stack pprof   -> <dir>/goroutine-<unixnano>.pprof
//   - runtime/metrics JSON    -> <dir>/runtime-metrics-<unixnano>.json
//
// and emits a one-shot summary log line. Captures are debounced (default 30
// minutes) to keep disk usage bounded even if the node sits over the
// threshold for a while.
//
// The package is self-contained: no cometbft, no cosmos-sdk logger, just
// stdlib. It does nothing on non-linux (no /proc) — RSS reads as 0 and the
// threshold is never tripped.
package memwatch

import (
	"bufio"
	"encoding/json"
	"fmt"
	stdlog "log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	envEnable      = "HEIMDALL_PRUNE_AUTOSNAP"
	envDir         = "HEIMDALL_PRUNE_AUTOSNAP_DIR"
	envThreshAbs   = "HEIMDALL_PRUNE_AUTOSNAP_THRESH_BYTES"   // absolute trigger
	envThreshMult  = "HEIMDALL_PRUNE_AUTOSNAP_THRESH_MULT"    // multiplicative trigger over baseline
	envBaseline    = "HEIMDALL_PRUNE_AUTOSNAP_BASELINE_BYTES" // baseline for multiplicative trigger
	envPollSeconds = "HEIMDALL_PRUNE_AUTOSNAP_POLL_SECS"
	envCooldownMin = "HEIMDALL_PRUNE_AUTOSNAP_COOLDOWN_MIN"

	defaultDir         = "/var/lib/heimdall/dumps"
	defaultPollSeconds = 5
	defaultCooldownMin = 30
	defaultThreshMult  = 2.0
)

// StartIfEnabled launches the watchdog goroutine when HEIMDALL_PRUNE_AUTOSNAP=1.
// Otherwise it is a no-op. Safe to call multiple times — only the first call
// starts the goroutine (subsequent calls are no-ops).
func StartIfEnabled() {
	if os.Getenv(envEnable) != "1" {
		return
	}
	if !started.CompareAndSwap(false, true) {
		return
	}
	go watchLoop()
}

var started atomic.Bool

func watchLoop() {
	cfg := readConfig()
	if err := os.MkdirAll(cfg.dir, 0o755); err != nil {
		stdlog.Printf("memwatch: cannot create dump dir %s: %v — disabling", cfg.dir, err)
		return
	}
	stdlog.Printf("memwatch: started dir=%s poll=%s thresh_abs=%dB baseline=%dB mult=%.2f cooldown=%s",
		cfg.dir, cfg.poll, cfg.threshAbs, cfg.baseline, cfg.threshMult, cfg.cooldown)

	ticker := time.NewTicker(cfg.poll)
	defer ticker.Stop()

	var lastCapture time.Time
	for range ticker.C {
		rss, ok := readVmRSSBytes()
		if !ok {
			continue
		}
		trip := false
		switch {
		case cfg.threshAbs > 0 && rss >= cfg.threshAbs:
			trip = true
		case cfg.baseline > 0 && cfg.threshMult > 0 && float64(rss) >= float64(cfg.baseline)*cfg.threshMult:
			trip = true
		}
		if !trip {
			continue
		}
		if time.Since(lastCapture) < cfg.cooldown {
			continue
		}
		lastCapture = time.Now()
		capture(cfg.dir, rss)
	}
}

type config struct {
	dir        string
	poll       time.Duration
	threshAbs  int64
	baseline   int64
	threshMult float64
	cooldown   time.Duration
}

func readConfig() config {
	c := config{
		dir:        getenvOr(envDir, defaultDir),
		poll:       time.Duration(getenvInt(envPollSeconds, defaultPollSeconds)) * time.Second,
		threshAbs:  int64(getenvInt(envThreshAbs, 0)),
		baseline:   int64(getenvInt(envBaseline, 0)),
		threshMult: getenvFloat(envThreshMult, defaultThreshMult),
		cooldown:   time.Duration(getenvInt(envCooldownMin, defaultCooldownMin)) * time.Minute,
	}
	if c.poll <= 0 {
		c.poll = time.Duration(defaultPollSeconds) * time.Second
	}
	return c
}

func capture(dir string, rss int64) {
	ts := time.Now().UnixNano()

	heapPath := filepath.Join(dir, fmt.Sprintf("heap-%d.pprof", ts))
	if err := writeHeapProfile(heapPath); err != nil {
		stdlog.Printf("memwatch: heap profile write failed: %v", err)
	}

	goroutinePath := filepath.Join(dir, fmt.Sprintf("goroutine-%d.pprof", ts))
	if err := writeGoroutineProfile(goroutinePath); err != nil {
		stdlog.Printf("memwatch: goroutine profile write failed: %v", err)
	}

	metricsPath := filepath.Join(dir, fmt.Sprintf("runtime-metrics-%d.json", ts))
	rm := writeRuntimeMetrics(metricsPath)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	stdlog.Printf("MEMWATCH-CAPTURE rss=%dB heap_alloc=%dB heap_inuse=%dB heap_idle=%dB heap_released=%dB stack_inuse=%dB sys=%dB num_gc=%d heap_dump=%s goroutine_dump=%s metrics_dump=%s rt_metrics=%v",
		rss, ms.HeapAlloc, ms.HeapInuse, ms.HeapIdle, ms.HeapReleased, ms.StackInuse, ms.Sys, ms.NumGC,
		heapPath, goroutinePath, metricsPath, rm)
}

func writeHeapProfile(path string) error {
	// Force a GC so the heap profile reflects steady-state allocation, not
	// transient garbage. The point of the capture is to see what's *retained*
	// during the spike.
	runtime.GC()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pprof.WriteHeapProfile(f)
}

func writeGoroutineProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	p := pprof.Lookup("goroutine")
	if p == nil {
		return fmt.Errorf("goroutine profile not available")
	}
	return p.WriteTo(f, 0) // proto format
}

// writeRuntimeMetrics dumps a small JSON-able summary of the runtime/metrics
// keys we care about. Returns the same map as a string for the log line.
func writeRuntimeMetrics(path string) string {
	keys := []string{
		"/memory/classes/heap/objects:bytes",
		"/memory/classes/heap/free:bytes",
		"/memory/classes/heap/released:bytes",
		"/memory/classes/heap/stacks:bytes",
		"/memory/classes/heap/unused:bytes",
		"/memory/classes/total:bytes",
		"/memory/classes/os-stacks:bytes",
		"/memory/classes/other:bytes",
		"/gc/heap/allocs:bytes",
		"/gc/heap/frees:bytes",
		"/gc/heap/live:bytes",
		"/gc/heap/objects:objects",
		"/gc/cycles/automatic:gc-cycles",
	}
	samples := make([]metrics.Sample, len(keys))
	for i, k := range keys {
		samples[i].Name = k
	}
	metrics.Read(samples)
	out := make(map[string]any, len(samples))
	for _, s := range samples {
		switch s.Value.Kind() {
		case metrics.KindUint64:
			out[s.Name] = s.Value.Uint64()
		case metrics.KindFloat64:
			out[s.Name] = s.Value.Float64()
		case metrics.KindFloat64Histogram:
			// histograms are bulky; skip in the summary
		default:
			out[s.Name] = nil
		}
	}
	if f, err := os.Create(path); err == nil {
		_ = json.NewEncoder(f).Encode(out)
		f.Close()
	}
	if b, err := json.Marshal(out); err == nil {
		return string(b)
	}
	return ""
}

// readVmRSSBytes reads VmRSS from /proc/self/status (linux). Returns
// (0, false) on non-linux or read errors.
func readVmRSSBytes() (int64, bool) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return 0, false
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, false
		}
		return kb * 1024, true
	}
	return 0, false
}

func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
