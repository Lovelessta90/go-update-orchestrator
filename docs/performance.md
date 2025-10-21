# Performance Optimization Guide

## Overview

This guide helps you identify, measure, and optimize performance issues in the Go Update Orchestrator codebase. It's designed for developers learning to spot "vibe code" and implement data-driven optimizations.

⚠️ **REALITY CHECK**: Most "optimizations" don't matter in production. Read the Reality Check section first!

---

## Reality Check: What Actually Matters

### The Harsh Truth About Micro-Optimizations

**Current baseline benchmark:**
```
BenchmarkPush-16    70,114 ops    50.1 μs/op    (local mock server)
```

**This number is MISLEADING because:**

1. **It's testing localhost** (no real network)
2. **Real devices are 100-1000x slower**
3. **Your code is 0.1% of total time**

### Real-World Performance Breakdown

**Updating a POS device over internet:**
```
Total time:        52ms
├─ Network latency: 50ms    (96.2% of time)  ← Can't optimize
├─ TLS handshake:   1.5ms   (2.9% of time)   ← Can't optimize
├─ Your code:       0.05ms  (0.1% of time)   ← Your "optimization"
└─ Device processing: 0.45ms (0.9% of time)  ← Can't control
```

**Your 30% optimization saves:**
- Your code: 50μs → 35μs = **15 microseconds**
- User sees: 52.050ms → 52.035ms = **0.03% faster**
- User perception: **NO DIFFERENCE**

### When Optimizations DO Matter

✅ **WORTH IT:**
1. **Algorithm improvements** (O(n²) → O(n))
   - Example: Sorting 10K devices: 100ms → 10ms
2. **Memory reduction** (prevents OOM crashes)
   - Example: Streaming 1GB file vs loading it all
3. **Concurrency** (better throughput)
   - Example: 10 concurrent → 100 concurrent
4. **Database queries** (N+1 problem)
   - Example: 10K queries → 1 query

❌ **NOT WORTH IT:**
1. **Micro-optimizations** (50μs → 35μs)
   - Saves 15 microseconds on a 50ms operation
2. **Premature optimization** (before profiling)
3. **Code complexity** for <5% gain
4. **Optimizing non-bottlenecks**

### Realistic Benchmark Results

```bash
$ go test -bench=BenchmarkRealisticScenario ./testing/integration/
```

```
Scenario              Time        Your Code  Network   What Matters
─────────────────────────────────────────────────────────────────────
Local mock           50 μs       100%       0%        ❌ Unrealistic
LAN (1ms)         1,175 μs        4%       96%       ✓ Some impact
Internet (50ms)  50,771 μs      0.1%      99.9%      ❌ Network-bound
```

**Key insight:** On real networks, **your code is noise**.

---

## Philosophy: Measure, Don't Guess

> **"Premature optimization is the root of all evil"** - Donald Knuth

But also:

> **"You can't improve what you don't measure"** - Peter Drucker

**The Balance**:
1. ✅ Write clean, correct code first
2. ✅ Establish baseline metrics
3. ✅ Profile to find real bottlenecks
4. ✅ Optimize hot paths only
5. ✅ Measure improvement

---

## Quick Start: Finding Bottlenecks

### 1. Run Benchmarks
```bash
go test -bench=. -benchmem ./pkg/delivery/http/
```

### 2. Profile CPU
```bash
go test -bench=BenchmarkPush -cpuprofile=cpu.prof ./pkg/delivery/http/
go tool pprof cpu.prof
```

In pprof:
```
(pprof) top10        # Show top 10 functions by time
(pprof) list Push    # Show line-by-line breakdown
(pprof) web          # Visual call graph (requires graphviz)
```

### 3. Profile Memory
```bash
go test -bench=BenchmarkPush -memprofile=mem.prof ./pkg/delivery/http/
go tool pprof mem.prof
```

In pprof:
```
(pprof) top10 -alloc_space    # Allocations
(pprof) top10 -inuse_space    # Current memory usage
(pprof) list Push
```

### 4. Trace Allocations
```bash
go test -bench=BenchmarkPush -trace=trace.out ./pkg/delivery/http/
go tool trace trace.out
```

---

## Spotting "Vibe Code" Anti-Patterns

### ❌ Anti-Pattern 1: String Concatenation in Loops
```go
// BAD: Multiple allocations
func buildURL(devices []Device) string {
    url := ""
    for _, d := range devices {
        url += d.Address + ","  // New allocation each iteration!
    }
    return url
}
```

**Cost**: O(n²) allocations

```go
// GOOD: Single allocation
func buildURL(devices []Device) string {
    var b strings.Builder
    b.Grow(len(devices) * 50)  // Pre-allocate
    for i, d := range devices {
        if i > 0 {
            b.WriteString(",")
        }
        b.WriteString(d.Address)
    }
    return b.String()
}
```

**Improvement**: O(1) allocations

### ❌ Anti-Pattern 2: Unbounded Goroutines
```go
// BAD: Can spawn 10,000 goroutines
func updateDevices(devices []Device) {
    for _, device := range devices {
        go updateDevice(device)  // No limit!
    }
}
```

**Cost**: Memory exhaustion, scheduler overhead

```go
// GOOD: Bounded worker pool
func updateDevices(devices []Device) {
    pool := NewWorkerPool(100)  // Max 100 concurrent
    pool.Start(ctx)
    defer pool.Stop()

    for _, device := range devices {
        pool.Submit(func(ctx context.Context) error {
            return updateDevice(device)
        })
    }
}
```

**Improvement**: Controlled resource usage

### ❌ Anti-Pattern 3: Loading Entire File Into Memory
```go
// BAD: 1GB file = 1GB RAM
func sendUpdate(device Device, filePath string) error {
    data, _ := os.ReadFile(filePath)  // Loads entire file!
    return client.Post(device.URL, bytes.NewReader(data))
}
```

**Cost**: O(file size) memory

```go
// GOOD: Streaming
func sendUpdate(device Device, filePath string) error {
    f, _ := os.Open(filePath)
    defer f.Close()
    return client.Post(device.URL, f)  // Streams!
}
```

**Improvement**: O(1) memory

### ❌ Anti-Pattern 4: Not Reusing HTTP Clients
```go
// BAD: New connection every time
func push(device Device, payload io.Reader) error {
    client := &http.Client{}  // Don't do this!
    resp, err := client.Post(device.URL, "application/octet-stream", payload)
    // ...
}
```

**Cost**: Connection overhead, no keep-alive

```go
// GOOD: Reuse client
type Delivery struct {
    client *http.Client  // Shared
}

func (d *Delivery) Push(device Device, payload io.Reader) error {
    resp, err := d.client.Post(device.URL, "application/octet-stream", payload)
    // ...
}
```

**Improvement**: Connection pooling, HTTP keep-alive

### ❌ Anti-Pattern 5: Allocating in Hot Paths
```go
// BAD: Allocates every call
func (d *Delivery) Push(device Device, payload io.Reader) error {
    buf := make([]byte, 8192)  // New allocation!
    io.CopyBuffer(dst, payload, buf)
}
```

**Cost**: GC pressure

```go
// GOOD: Reuse buffers
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 8192)
    },
}

func (d *Delivery) Push(device Device, payload io.Reader) error {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)
    io.CopyBuffer(dst, payload, buf)
}
```

**Improvement**: Fewer allocations, less GC

---

## Optimization Checklist

### Before Optimizing
- [ ] Does this code work correctly?
- [ ] Are there tests?
- [ ] Have I profiled it?
- [ ] Is this actually a bottleneck?
- [ ] What's the current benchmark?

### During Optimization
- [ ] Did I keep the code readable?
- [ ] Did I add comments explaining trade-offs?
- [ ] Did tests still pass?
- [ ] Did I benchmark before and after?
- [ ] Is the improvement significant (>10%)?

### After Optimization
- [ ] Did I document the change?
- [ ] Did I compare with baseline?
- [ ] Did I check for edge cases?
- [ ] Did I update docs if API changed?
- [ ] Did I commit with benchstat output?

---

## Profiling Workflow

### Step 1: Identify Hotspot
```bash
go test -bench=BenchmarkPush -cpuprofile=cpu.prof ./pkg/delivery/http/
go tool pprof -http=:8080 cpu.prof
```

Look for:
- Functions taking >5% CPU time
- Tight loops
- Many small allocations

### Step 2: Analyze Allocations
```bash
go test -bench=BenchmarkPush -memprofile=mem.prof ./pkg/delivery/http/
go tool pprof -http=:8080 mem.prof
```

Look for:
- Functions allocating >1KB per call
- String conversions
- Map/slice growth

### Step 3: Trace Execution
```bash
go test -bench=BenchmarkPush -trace=trace.out ./pkg/delivery/http/
go tool trace trace.out
```

Look for:
- Goroutine blocking
- GC pauses
- Network latency

### Step 4: Optimize
Make ONE change at a time.

### Step 5: Benchmark
```bash
go test -bench=BenchmarkPush -benchmem -count=10 ./pkg/delivery/http/ > new.txt
benchstat baseline.txt new.txt
```

### Step 6: Decide
- Improvement >10%? → Keep it
- Improvement <10%? → Consider readability trade-off
- No improvement? → Revert

---

## Common Optimization Techniques

### 1. Buffer Pooling (High Impact)
```go
var pool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 8192)
    },
}

// Use
buf := pool.Get().([]byte)
defer pool.Put(buf)
```

**When**: Frequent allocation/deallocation of same-size buffers
**Impact**: 20-40% reduction in allocations

### 2. String Interning (Medium Impact)
```go
var headerKeys = map[string]string{
    "content-type": "Content-Type",
    "authorization": "Authorization",
}

func getHeader(key string) string {
    if canonical, ok := headerKeys[strings.ToLower(key)]; ok {
        return canonical
    }
    return http.CanonicalHeaderKey(key)
}
```

**When**: Same strings allocated repeatedly
**Impact**: 10-20% reduction in string allocations

### 3. Pre-allocation (Low-Medium Impact)
```go
// Instead of:
devices := []Device{}

// Do:
devices := make([]Device, 0, expectedCount)
```

**When**: You know approximate size
**Impact**: 5-15% reduction in allocations

### 4. Inline Small Functions (Low Impact)
```go
//go:inline
func isOnline(d Device) bool {
    return d.Status == DeviceOnline
}
```

**When**: Tiny functions called in hot paths
**Impact**: 2-5% latency reduction

### 5. Batch Operations (High Impact)
```go
// Instead of:
for _, device := range devices {
    db.Update(device)  // N database calls
}

// Do:
db.BatchUpdate(devices)  // 1 database call
```

**When**: Multiple network/disk operations
**Impact**: 50-90% latency reduction

---

## Benchmark Comparison

### Using benchstat
```bash
# Install
go install golang.org/x/perf/cmd/benchstat@latest

# Save baseline
go test -bench=. -benchmem -count=10 ./pkg/delivery/http/ > baseline.txt

# Make changes...

# Save new results
go test -bench=. -benchmem -count=10 ./pkg/delivery/http/ > optimized.txt

# Compare
benchstat baseline.txt optimized.txt
```

### Reading benchstat Output
```
name       old time/op    new time/op    delta
Push-16      50.1µs ± 2%    35.2µs ± 1%  -29.74%  (p=0.000 n=10+10)
             ^^^^^^         ^^^^^^         ^^^^^^
             Before         After          Change

name       old alloc/op   new alloc/op   delta
Push-16      7.45kB ± 0%    5.12kB ± 0%  -31.27%  (p=0.000 n=10+10)
```

**Interpreting**:
- `±2%` - Variance (lower is more stable)
- `-29.74%` - Percent improvement (negative = faster/less)
- `p=0.000` - Statistical significance (< 0.05 = significant)
- `n=10+10` - Sample count

**Good Change**:
- Time: -20% or more
- Allocations: -30% or more
- Low variance (±5% or less)
- Significant p-value

---

## Performance Goals

### Current Baseline (See BASELINE_METRICS.md)
```
Push:        50.1 µs/op,  7.5 KB/op,  88 allocs/op
Verify:     115.4 µs/op, 17.9 KB/op, 126 allocs/op
Throughput:  ~20K ops/s (push)
```

### Target Goals
```
Push:        35-40 µs/op,  5-5.5 KB/op,  50-60 allocs/op
Verify:      80-90 µs/op, 12-14 KB/op,  80-90 allocs/op
Throughput:  ~25K-30K ops/s (push)
```

### Stretch Goals
```
Push:        <30 µs/op,  <4 KB/op,  <40 allocs/op
Verify:      <70 µs/op, <10 KB/op,  <70 allocs/op
Throughput:  >35K ops/s (push)
```

---

## Red Flags: When NOT to Optimize

### ❌ Don't Optimize If...
1. **No measurement** - "Feels slow" isn't a metric
2. **Not a bottleneck** - Function uses <1% CPU
3. **Sacrifices readability** - Complex code for <5% gain
4. **Breaks abstractions** - Tight coupling for speed
5. **Premature** - Feature isn't finished yet

### ✅ DO Optimize If...
1. **Profiled** - Data shows it's a hotspot
2. **Measurable** - Benchmarks show impact
3. **Significant** - >10% improvement
4. **Maintainable** - Code stays readable
5. **Tested** - All tests still pass

---

## Example: Optimizing Push()

### Before (Baseline)
```go
func (d *Delivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
    url := device.Address + d.config.UpdateEndpoint  // String concatenation

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, payload)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/octet-stream")  // Allocates

    for key, value := range d.config.Headers {
        req.Header.Set(key, value)  // More allocations
    }

    req.Header.Set("X-Device-ID", device.ID)
    req.Header.Set("X-Device-Name", device.Name)

    resp, err := d.client.Do(req)
    // ... rest of function
}
```

**Benchmark**: 50.1 µs/op, 7.5 KB/op, 88 allocs/op

### After (Optimized)
```go
var urlBuilder = sync.Pool{
    New: func() interface{} {
        return &strings.Builder{}
    },
}

func (d *Delivery) Push(ctx context.Context, device core.Device, payload io.Reader) error {
    // Reuse string builder
    sb := urlBuilder.Get().(*strings.Builder)
    sb.Reset()
    sb.WriteString(device.Address)
    sb.WriteString(d.config.UpdateEndpoint)
    url := sb.String()
    urlBuilder.Put(sb)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, payload)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    // Pre-set all headers at once (fewer map operations)
    req.Header = http.Header{
        "Content-Type":  {"application/octet-stream"},
        "X-Device-ID":   {device.ID},
        "X-Device-Name": {device.Name},
    }

    // Add custom headers
    for key, value := range d.config.Headers {
        req.Header.Set(key, value)
    }

    resp, err := d.client.Do(req)
    // ... rest of function
}
```

**Benchmark**: 38.2 µs/op, 5.8 KB/op, 62 allocs/op

**Improvement**: -23.8% latency, -22.8% memory, -29.5% allocations

---

## Learning Path

### Beginner
1. Read [BASELINE_METRICS.md](../BASELINE_METRICS.md)
2. Run benchmarks: `go test -bench=. -benchmem`
3. Learn pprof: `go tool pprof -http=:8080 cpu.prof`
4. Try one optimization from this guide
5. Measure with benchstat

### Intermediate
1. Profile real workloads (not just benchmarks)
2. Identify allocation hotspots
3. Implement buffer pooling
4. Optimize string handling
5. Reduce interface boxing

### Advanced
1. Write custom allocators
2. Use assembly for critical paths
3. Optimize cache locality
4. Reduce system calls
5. Zero-allocation parsing

---

## Resources

### Official Go Documentation
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Diagnostics](https://go.dev/doc/diagnostics)
- [Performance](https://github.com/golang/go/wiki/Performance)

### Deep Dives
- [High Performance Go Workshop](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)
- [Allocation Efficiency](https://segment.com/blog/allocation-efficiency-in-high-performance-go-services/)
- [sync.Pool Deep Dive](https://victoriametrics.com/blog/go-sync-pool/)

### Tools
- [pprof](https://github.com/google/pprof)
- [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [go-torch](https://github.com/uber/go-torch) (flame graphs)

---

## Summary

1. **Measure first** - Profile before optimizing
2. **One change at a time** - Isolate improvements
3. **Benchmark everything** - Before and after
4. **Keep it simple** - Readability matters
5. **Document trade-offs** - Future you will thank you

Remember: **The best optimization is the one you can measure and maintain.**
