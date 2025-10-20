# POS System Update Example

This example demonstrates **real-world update strategies** for managing a fleet of Point-of-Sale (POS) terminals across multiple retail locations.

## The Problem

You're a retail company with **10 stores** across the United States, each with 1-2 POS terminals (10 devices total). You need to:

- Push critical security patches immediately
- Schedule major updates during low-traffic hours (2 AM - 4 AM)
- Test new firmware on a small subset before full rollout
- Handle devices that are intermittently connected
- Target specific regions or firmware versions

## Device Fleet Overview

Our sample fleet includes:

- **6 Online devices** - Currently connected and reachable
- **3 Offline devices** - Not currently connected (stores closed, network issues)
- **1 Unknown device** - Never seen or status unclear

Devices span multiple regions: Northeast, West, Midwest, Southeast, Northwest, Southwest, Mountain.

## Running the Example

```bash
cd examples/pos_system
go run main.go
```

## Update Scenarios Demonstrated

### 1. Immediate Update (Critical Security Patch)

**Use Case**: CVE announced, need to patch ALL online devices RIGHT NOW

**Strategy**: `StrategyImmediate`

```go
update := core.Update{
    Strategy: core.StrategyImmediate,
    DeviceFilter: &core.Filter{
        Status: &core.DeviceOnline, // Only online devices
    },
}
```

**What happens**:
- Orchestrator immediately pushes to all 6 online devices
- 3 offline devices are skipped (you'll need on-connect strategy)
- No scheduling, no delays

**Real-world use**: Critical security patches, urgent bug fixes

---

### 2. Scheduled Maintenance Window

**Use Case**: Major firmware update with potential downtime

**Strategy**: `StrategyScheduled`

```go
scheduledTime := time.Date(2024, 10, 21, 2, 0, 0, 0, time.Local) // 2 AM
windowEnd := scheduledTime.Add(2 * time.Hour) // 4 AM

update := core.Update{
    Strategy:    core.StrategyScheduled,
    ScheduledAt: &scheduledTime,
    WindowStart: &scheduledTime,
    WindowEnd:   &windowEnd,
}
```

**What happens**:
- Orchestrator waits until 2 AM
- At 2 AM, checks which devices are online
- Pushes updates to online devices
- Stops accepting new updates at 4 AM

**Real-world use**: Feature releases, major firmware upgrades, anything that might cause brief downtime

---

### 3. Progressive Rollout (Risk Mitigation)

**Use Case**: New untested firmware, want to minimize blast radius

**Strategy**: `StrategyProgressive`

```go
phases := []core.RolloutPhase{
    {
        Name:        "Canary",
        Percentage:  10,  // 1 device (10% of 10)
        WaitTime:    24 * time.Hour,
        SuccessRate: 95,  // Must be 95%+ successful
    },
    {
        Name:        "Phase 1",
        Percentage:  50,  // 5 devices
        WaitTime:    12 * time.Hour,
        SuccessRate: 90,
    },
    {
        Name:        "Phase 2",
        Percentage:  100, // All remaining
        WaitTime:    0,
        SuccessRate: 85,
    },
}

update := core.Update{
    Strategy:      core.StrategyProgressive,
    RolloutPhases: phases,
}
```

**What happens**:
1. **Day 1**: Update 1 device (canary), monitor for 24 hours
2. **Day 2**: If 95%+ success, update 4 more devices (total 5)
3. **Day 2.5**: If still 90%+ success, update remaining 5
4. If any phase fails threshold, STOP and rollback

**Real-world use**: Major version updates, architectural changes, risky deployments

---

### 4. On-Connect Strategy (Offline Devices)

**Use Case**: You have 3 offline devices (stores closed, network down)

**Strategy**: `StrategyOnConnect`

```go
update := core.Update{
    Strategy: core.StrategyOnConnect,
}
```

**What happens**:
- Update is registered in the system as "pending"
- When a device connects (sends heartbeat), orchestrator checks for pending updates
- Update is immediately pushed to the newly connected device
- No manual intervention required

**Real-world use**:
- Mobile POS systems (food trucks, pop-up stores)
- Seasonal locations (ski resorts, beach shops)
- Devices with unreliable connectivity
- International fleets with timezone differences

**How devices "connect"**:
```
Device → Heartbeat every 5 min → Orchestrator
                                  ↓
                           "Any updates for me?"
                                  ↓
                           Yes → Push update
```

---

### 5. Region-Based Update

**Use Case**: Tax law change in Northeast states (NY, MA)

**Strategy**: `StrategyImmediate` with region filter

```go
update := core.Update{
    Strategy: core.StrategyImmediate,
    DeviceFilter: &core.Filter{
        Tags: map[string]string{
            "region": "northeast",
        },
    },
}
```

**What happens**:
- Only devices tagged with `region: northeast` receive update
- 3 devices in NYC and Boston get the update
- All other regions unaffected

**Real-world use**:
- Regional tax updates
- State-specific compliance
- Language packs for specific regions
- Beta testing in select markets

---

### 6. Firmware Version Upgrade

**Use Case**: Upgrade all devices running old firmware (v2.0.x, v2.1.x)

**Strategy**: `StrategyScheduled` with firmware filter

```go
update := core.Update{
    Strategy: core.StrategyScheduled,
    DeviceFilter: &core.Filter{
        MaxFirmware: "2.2.0", // Devices with firmware < v2.2.0
    },
}
```

**What happens**:
- Only devices with firmware v2.0.x or v2.1.x are updated
- Devices already on v2.3.0+ are skipped
- Keeps fleet standardized

**Real-world use**:
- End-of-life firmware versions
- Security vulnerabilities in specific versions
- Feature parity across fleet

---

## Device Registry: The Answer to Your Question

### How Companies Manage Updates

**Yes, companies maintain a FULL REGISTRY of all devices**, even offline ones:

```json
{
  "id": "pos-miami-001",
  "status": "offline",
  "last_seen": "2024-10-19T14:23:00Z",  // 24 hours ago
  "firmware_version": "2.3.0",
  "location": "Miami, FL",
  "metadata": {
    "store_id": "MIA-001",
    "priority": "low"
  }
}
```

**Why maintain a full registry?**

1. **Compliance**: "Which devices are running EOL firmware?"
2. **Support**: "Is device XYZ even online?"
3. **Analytics**: "What's our fleet firmware distribution?"
4. **Security**: "Which devices haven't checked in for 30 days?" (compromised?)
5. **Planning**: "How many devices need updates?"

**How updates work with offline devices**:

```
┌─────────────────┐
│ 10,000 Devices  │
├─────────────────┤
│ 7,000 Online    │ ← Update immediately
│ 3,000 Offline   │ ← Schedule for on-connect
└─────────────────┘
```

**Three common patterns**:

1. **Push to online, queue for offline** (most common)
   - Update online devices now
   - When offline devices connect, they pull pending updates

2. **Scheduled window** (POS systems, kiosks)
   - Devices know to be online at 2 AM
   - Server pushes during that window
   - Devices that miss window get update next day

3. **Pull-based** (very intermittent connectivity)
   - Devices check in when they connect
   - "Any updates for me?" → Server responds
   - Common for IoT, mobile devices

## Key Insights

### Device Lifecycle
```
Register → Online → Offline → Online → Update → Offline → Decommission
           ↑         ↓         ↑         ↓
           └─────────┴─────────┴─────────┘
              Heartbeat every 5 min
```

### Update State Machine
```
Pending → Scheduled → In Progress → Completed
   ↓                       ↓
Cancelled              Failed → Retry → Completed
```

### Real-World Considerations

1. **Heartbeat Mechanism**
   - Devices ping server every 5 minutes
   - Updates `last_seen` timestamp
   - Server knows online/offline status

2. **Network Reliability**
   - Devices may go offline mid-update
   - Need retry logic
   - Need resume capability for large files

3. **Rollback Support**
   - If update fails, rollback to previous firmware
   - Keep previous version on device
   - Automatic or manual rollback

4. **Bandwidth Management**
   - Don't update 10k devices simultaneously
   - Bounded concurrency (e.g., 100 at a time)
   - Rate limiting per region

## Next Steps

This example demonstrates the **API design** for these scenarios. To make it work:

1. Implement `Orchestrator.ExecuteUpdate()` with strategy handling
2. Implement `ProgressTracker` for monitoring
3. Implement device heartbeat mechanism
4. Add retry logic for failed updates
5. Implement progressive rollout logic

See the main README for development roadmap.
