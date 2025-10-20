# Device Management Strategies

## The Core Question

**"How does a company manage updates for 10,000 vehicles across the U.S. when they're not connected 100% of the time?"**

## TL;DR Answer

**Yes, companies maintain a FULL REGISTRY of ALL devices** (even offline ones), and use a combination of strategies:

1. **Push to online devices immediately**
2. **Queue updates for offline devices** (delivered when they connect)
3. **Scheduled maintenance windows** (devices know to be online at specific times)
4. **Pull-based updates** (devices check for updates when they connect)

---

## Real-World Device Management

### Complete Device Registry

Every company with a device fleet maintains a **complete registry** that tracks:

```
┌─────────────────────────────────────────────┐
│            Device Registry                  │
├─────────────────────────────────────────────┤
│ Device ID:        vehicle-12345             │
│ Status:           offline                   │
│ Last Seen:        2024-10-19 14:23:00 UTC   │
│ Firmware:         v2.3.1                    │
│ Location:         Seattle, WA               │
│ Customer:         Fleet Operator XYZ        │
│ Warranty:         2026-01-15                │
│ Tags:             { region: northwest }     │
└─────────────────────────────────────────────┘
```

### Why Maintain a Full Registry?

1. **Compliance & Auditing**
   - "Which vehicles have firmware < v2.0?" (security vulnerability)
   - "Show all devices in California" (CARB compliance)
   - "What's our fleet-wide firmware distribution?"

2. **Support & Operations**
   - "Is vehicle XYZ even registered?"
   - "When did this vehicle last connect?"
   - "Which customer owns this device?"

3. **Security**
   - Detect compromised devices (offline for 30+ days)
   - Track unauthorized devices
   - Monitor for anomalies

4. **Planning**
   - "How many devices need updates?"
   - "What's our rollout timeline?"
   - Revenue forecasting (connected vs. total devices)

---

## Update Distribution Strategies

### Strategy 1: Immediate Push (Online Devices)

**Use Case**: Critical security patch

```
Fleet: 10,000 vehicles
├─ 7,000 Online   ← Push update NOW
└─ 3,000 Offline  ← Queue for later
```

**Implementation**:
```go
update := core.Update{
    Strategy: core.StrategyImmediate,
    DeviceFilter: &core.Filter{
        Status: &core.DeviceOnline,
    },
}
```

**What happens**:
- Orchestrator pushes to 7,000 online vehicles immediately
- 3,000 offline vehicles: update queued as "pending"
- When offline vehicle connects → automatically receives update

**Real-world timing**:
- Online vehicles: Updated in 10-30 minutes (batched, 100 concurrent)
- Offline vehicles: Updated when they connect (hours to days)

---

### Strategy 2: Scheduled Maintenance Window

**Use Case**: Major firmware update with potential downtime

```
Maintenance Window: 2 AM - 4 AM (local time zones)
```

**Implementation**:
```go
scheduledTime := time.Date(2024, 10, 21, 2, 0, 0, 0, time.Local)
windowEnd := scheduledTime.Add(2 * time.Hour)

update := core.Update{
    Strategy:    core.StrategyScheduled,
    WindowStart: &scheduledTime,
    WindowEnd:   &windowEnd,
}
```

**What happens**:
1. **Before 2 AM**: Update is in "scheduled" status
2. **At 2 AM**: Orchestrator wakes up, checks which devices are online
3. **2:00 - 4:00 AM**: Pushes updates to online devices
4. **After 4 AM**: Window closes, remaining devices get on-connect update
5. **Devices that miss window**: Updated next night or when they connect

**Common for**:
- POS systems (restaurants close at midnight)
- Digital signage (low-traffic hours)
- Fleet vehicles (parked overnight at depot)

---

### Strategy 3: Progressive Rollout (Risk Mitigation)

**Use Case**: Untested firmware, want to minimize blast radius

```
Phase 1: Canary  → 100 vehicles (1%)     → Wait 24h → Verify 95%+ success
Phase 2: Early   → 1,000 vehicles (10%)  → Wait 12h → Verify 90%+ success
Phase 3: Full    → 8,900 vehicles (89%)  → Complete
```

**Implementation**:
```go
phases := []core.RolloutPhase{
    {Name: "Canary", Percentage: 1, WaitTime: 24h, SuccessRate: 95},
    {Name: "Early", Percentage: 10, WaitTime: 12h, SuccessRate: 90},
    {Name: "Full", Percentage: 100, WaitTime: 0, SuccessRate: 85},
}

update := core.Update{
    Strategy: core.StrategyProgressive,
    RolloutPhases: phases,
}
```

**Safety features**:
- If Phase 1 success rate < 95% → HALT, rollback
- Manual approval between phases (optional)
- Real-time monitoring dashboard
- Can pause/cancel at any phase

**Real-world examples**:
- Tesla Over-The-Air updates (progressive rollout by VIN)
- Microsoft Windows updates (rings: Insiders → Release Preview → General)
- Mobile app updates (Google Play staged rollouts)

---

### Strategy 4: On-Connect (Pull-Based)

**Use Case**: Vehicles with intermittent connectivity

```
Device connects → "Any updates for me?" → Server responds → Push update
```

**Implementation**:
```go
update := core.Update{
    Strategy: core.StrategyOnConnect,
}
```

**Device heartbeat mechanism**:
```
Every 5 minutes (when connected):
┌────────┐                        ┌────────────┐
│ Device │ ─── Heartbeat ──────→  │ Server     │
│        │ ← "Update pending" ──  │            │
│        │ ─── Pull update ─────→ │            │
└────────┘                        └────────────┘
```

**Perfect for**:
- Vehicles that aren't always connected (rural areas, international)
- Mobile devices (phones, tablets)
- IoT devices with unreliable connectivity
- Seasonal equipment (ski resort gondolas, beach kiosks)

---

## Real-World Implementation Details

### How Devices Report Status

**Heartbeat Protocol** (most common):
```
Device → Server (every 5 minutes):
{
    "device_id": "vehicle-12345",
    "firmware_version": "2.3.1",
    "timestamp": "2024-10-20T14:23:00Z",
    "location": { "lat": 47.6062, "lon": -122.3321 },
    "battery": 87,
    "signal_strength": -65
}

Server → Device:
{
    "pending_updates": ["update-001"],
    "next_heartbeat": 300  // seconds
}
```

**Status transitions**:
```
Device starts → "unknown"
First heartbeat → "online"
No heartbeat for 10 min → "offline"
No heartbeat for 30 days → "abandoned" (alert ops team)
```

### Update Delivery Flow

**For 10,000 vehicle fleet**:

```
1. Create Update Job
   └─ Target: All vehicles with firmware < v2.3.0
   └─ Strategy: Progressive rollout
   └─ Total devices: 10,000

2. Phase 1: Canary (100 vehicles)
   ├─ Filter: 100 online vehicles, random selection
   ├─ Push update (10 concurrent)
   ├─ Monitor for 24 hours
   └─ Success rate: 98% ✓

3. Phase 2: Early Adopters (1,000 vehicles)
   ├─ Filter: Next 1,000 online vehicles
   ├─ Push update (100 concurrent)
   ├─ Monitor for 12 hours
   └─ Success rate: 95% ✓

4. Phase 3: Full Rollout (8,900 vehicles)
   ├─ Push to all remaining vehicles
   ├─ Online: 6,230 → updated immediately
   ├─ Offline: 2,670 → queued for on-connect
   └─ Over 7 days: All offline devices connect and update
```

### Bandwidth Management

**Problem**: Don't want 10,000 devices downloading simultaneously

**Solution 1: Bounded Concurrency**
```go
config := &orchestrator.Config{
    MaxConcurrent: 100,  // Only 100 downloads at once
}
```

**Solution 2: Rate Limiting**
- Per region: Max 50 Mbps
- Per device: Throttle to 1 Mbps
- Time-based: More bandwidth during off-peak hours

**Solution 3: CDN + Chunking**
```
Central Server → Regional CDN → Edge Cache → Device
     |              |               |
  Update        Replicate       Chunked
  available     to regions      download
```

---

## Connectivity Patterns by Industry

### 1. Fleet Vehicles (Tesla, UPS, Trucking)
- **Connectivity**: 50-70% online at any time
- **Update strategy**: On-connect + scheduled (overnight when parked)
- **Heartbeat**: Every 5 minutes when ignition on
- **Typical update duration**: 10-30 minutes

### 2. POS Systems (Retail, Restaurants)
- **Connectivity**: 90%+ online during business hours
- **Update strategy**: Scheduled (2 AM - 4 AM)
- **Heartbeat**: Every 1 minute during business hours
- **Typical update duration**: 5-15 minutes

### 3. Digital Signage (Airports, Malls)
- **Connectivity**: 95%+ online (wired connection)
- **Update strategy**: Immediate or scheduled (low-traffic hours)
- **Heartbeat**: Every 5 minutes
- **Typical update duration**: 2-10 minutes

### 4. IoT Sensors (Smart Home, Industrial)
- **Connectivity**: 20-40% online (battery-powered, intermittent)
- **Update strategy**: On-connect (aggressive)
- **Heartbeat**: Every 15-60 minutes (battery-dependent)
- **Typical update duration**: 1-5 minutes

### 5. Medical Devices (Hospitals, Clinics)
- **Connectivity**: 80%+ online (critical systems)
- **Update strategy**: Scheduled + approval required
- **Heartbeat**: Every 30 seconds (critical monitoring)
- **Typical update duration**: 10-30 minutes with rollback

---

## Network Architecture

### Centralized Model
```
                ┌──────────────┐
                │  Orchestrator │
                │   (Central)   │
                └───────┬───────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌─────────┐     ┌─────────┐     ┌─────────┐
   │ Region  │     │ Region  │     │ Region  │
   │  West   │     │  East   │     │ Central │
   └────┬────┘     └────┬────┘     └────┬────┘
        │               │               │
    ┌───┴───┐       ┌───┴───┐       ┌───┴───┐
    ▼       ▼       ▼       ▼       ▼       ▼
  Dev1    Dev2    Dev3    Dev4    Dev5    Dev6
```

**Pros**: Simple, centralized control
**Cons**: Single point of failure, bandwidth bottleneck

### Edge/Hybrid Model
```
                ┌──────────────┐
                │ Orchestrator │
                │ (Coordination)│
                └───────┬───────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌─────────┐     ┌─────────┐     ┌─────────┐
   │  Edge   │     │  Edge   │     │  Edge   │
   │ Server  │     │ Server  │     │ Server  │
   │ + Cache │     │ + Cache │     │ + Cache │
   └────┬────┘     └────┬────┘     └────┬────┘
        │               │               │
    ┌───┴───┐       ┌───┴───┐       ┌───┴───┐
    ▼       ▼       ▼       ▼       ▼       ▼
  Dev1    Dev2    Dev3    Dev4    Dev5    Dev6
```

**Pros**: Better bandwidth, resilient
**Cons**: More complex, cache invalidation

---

## Key Insights

### 1. Always Maintain Full Registry
Even for offline devices. You need to know:
- What's deployed
- What's missing
- What needs updating

### 2. Expect 30-50% Offline
For most fleets (vehicles, mobile devices), only 30-70% are online at any time.

### 3. Use Multiple Strategies
- Immediate for critical patches
- Scheduled for major updates
- On-connect for offline devices
- Progressive for risky updates

### 4. Network Matters
- Wired (POS, signage): 95%+ online, reliable
- Cellular (vehicles, IoT): 50-70% online, unreliable
- WiFi (mobile devices): 30-50% online, variable

### 5. Time Zones Are Real
10K devices across U.S. = 4 time zones. Schedule "2 AM" means:
- 2 AM Pacific (L.A.)
- 2 AM Mountain (Denver)
- 2 AM Central (Chicago)
- 2 AM Eastern (NYC)

Not "2 AM UTC" for everyone.

---

## Implementation Checklist

- [ ] Device registry with full metadata
- [ ] Heartbeat mechanism (device → server)
- [ ] Online/offline status tracking
- [ ] Pending update queue
- [ ] Multiple update strategies
- [ ] Bounded concurrency (don't overwhelm network)
- [ ] Retry logic (transient failures)
- [ ] Rollback support (failed updates)
- [ ] Progress tracking (per-device)
- [ ] Time zone handling
- [ ] CDN/edge caching (optional, for scale)
- [ ] Metrics and alerting

---

## Further Reading

- [Architecture](architecture.md) - System design
- [POS Example](../examples/pos_system/README.md) - Working code
- [Interfaces](interfaces.md) - API contracts
