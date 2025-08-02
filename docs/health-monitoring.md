# Health Monitoring - Disk Space Implementation Guide

## Overview

The DataScrapexter health monitoring system provides comprehensive health checks but does not include disk space monitoring out of the box due to platform-specific implementation requirements.

## Platform-Specific Implementations

### Unix/Linux Implementation

```go
package monitoring

import (
    "context"
    "fmt"
    "golang.org/x/sys/unix"
)

func diskSpace(path string) (free, total uint64, err error) {
    var stat unix.Statfs_t
    err = unix.Statfs(path, &stat)
    if err != nil {
        return 0, 0, err
    }
    
    free = stat.Bavail * uint64(stat.Bsize)
    total = stat.Blocks * uint64(stat.Bsize)
    return free, total, nil
}

// DiskSpaceHealthCheck creates a Unix/Linux disk space health check
func DiskSpaceHealthCheck(path string, maxUsagePercent float64) *HealthCheck {
    return &HealthCheck{
        Name:     "disk_space",
        Critical: false,
        Enabled:  true,
        CheckFunc: func(ctx context.Context) HealthCheckResult {
            free, total, err := diskSpace(path)
            if err != nil {
                return HealthCheckResult{
                    Status:  HealthStatusUnhealthy,
                    Message: "Failed to check disk space",
                    Error:   err,
                }
            }
            
            used := total - free
            percent := float64(used) / float64(total) * 100
            
            metadata := map[string]interface{}{
                "path":            path,
                "total_bytes":     total,
                "used_bytes":      used,
                "free_bytes":      free,
                "usage_percent":   percent,
            }
            
            if percent > maxUsagePercent {
                return HealthCheckResult{
                    Status:   HealthStatusDegraded,
                    Message:  fmt.Sprintf("High disk usage: %.1f%%", percent),
                    Metadata: metadata,
                }
            }
            
            return HealthCheckResult{
                Status:   HealthStatusHealthy,
                Message:  fmt.Sprintf("Disk usage normal: %.1f%%", percent),
                Metadata: metadata,
            }
        },
    }
}
```

### Windows Implementation

```go
package monitoring

import (
    "context"
    "fmt"
    "syscall"
    "unsafe"
    "golang.org/x/sys/windows"
)

func diskSpaceWindows(path string) (free, total uint64, err error) {
    kernel32 := windows.MustLoadDLL("kernel32.dll")
    getDiskFreeSpaceEx := kernel32.MustFindProc("GetDiskFreeSpaceExW")
    
    pathPtr, err := syscall.UTF16PtrFromString(path)
    if err != nil {
        return 0, 0, err
    }
    
    var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
    
    r1, _, err := getDiskFreeSpaceEx.Call(
        uintptr(unsafe.Pointer(pathPtr)),
        uintptr(unsafe.Pointer(&freeBytesAvailable)),
        uintptr(unsafe.Pointer(&totalNumberOfBytes)),
        uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
    )
    
    if r1 == 0 {
        return 0, 0, err
    }
    
    return freeBytesAvailable, totalNumberOfBytes, nil
}

// DiskSpaceHealthCheckWindows creates a Windows disk space health check
func DiskSpaceHealthCheckWindows(path string, maxUsagePercent float64) *HealthCheck {
    return &HealthCheck{
        Name:     "disk_space",
        Critical: false,
        Enabled:  true,
        CheckFunc: func(ctx context.Context) HealthCheckResult {
            free, total, err := diskSpaceWindows(path)
            if err != nil {
                return HealthCheckResult{
                    Status:  HealthStatusUnhealthy,
                    Message: "Failed to check disk space",
                    Error:   err,
                }
            }
            
            used := total - free
            percent := float64(used) / float64(total) * 100
            
            if percent > maxUsagePercent {
                return HealthCheckResult{
                    Status:  HealthStatusDegraded,
                    Message: fmt.Sprintf("High disk usage: %.1f%%", percent),
                }
            }
            
            return HealthCheckResult{
                Status:  HealthStatusHealthy,
                Message: fmt.Sprintf("Disk usage normal: %.1f%%", percent),
            }
        },
    }
}
```

### Cross-Platform Implementation (Recommended)

```go
package monitoring

import (
    "context"
    "fmt"
    "github.com/shirou/gopsutil/v3/disk"
)

// DiskSpaceHealthCheckCrossPlatform creates a cross-platform disk space health check
// This is the recommended approach for most applications
func DiskSpaceHealthCheckCrossPlatform(path string, maxUsagePercent float64) *HealthCheck {
    return &HealthCheck{
        Name:     "disk_space",
        Critical: false,
        Enabled:  true,
        CheckFunc: func(ctx context.Context) HealthCheckResult {
            usage, err := disk.Usage(path)
            if err != nil {
                return HealthCheckResult{
                    Status:  HealthStatusUnhealthy,
                    Message: "Failed to check disk space",
                    Error:   err,
                }
            }
            
            percent := usage.UsedPercent
            
            metadata := map[string]interface{}{
                "path":            path,
                "total_bytes":     usage.Total,
                "used_bytes":      usage.Used,
                "free_bytes":      usage.Free,
                "usage_percent":   percent,
                "filesystem":      usage.Fstype,
            }
            
            if percent > maxUsagePercent {
                return HealthCheckResult{
                    Status:   HealthStatusDegraded,
                    Message:  fmt.Sprintf("High disk usage: %.1f%%", percent),
                    Metadata: metadata,
                }
            }
            
            return HealthCheckResult{
                Status:   HealthStatusHealthy,
                Message:  fmt.Sprintf("Disk usage normal: %.1f%%", percent),
                Metadata: metadata,
            }
        },
    }
}
```

## Usage Examples

### Registering Disk Space Health Check

```go
// Using cross-platform implementation (recommended)
healthManager := NewHealthManager(HealthConfig{})

// Monitor root filesystem with 80% threshold
diskCheck := DiskSpaceHealthCheckCrossPlatform("/", 80.0)
healthManager.RegisterCheck(diskCheck)

// Monitor specific application data directory
dataCheck := DiskSpaceHealthCheckCrossPlatform("/var/lib/datascrapexter", 85.0)
healthManager.RegisterCheck(dataCheck)

// Start health monitoring
ctx := context.Background()
healthManager.Start(ctx)
```

### Multiple Mount Points

```go
// Monitor multiple mount points
mountPoints := []struct {
    path      string
    threshold float64
    critical  bool
}{
    {"/", 90.0, true},           // Root filesystem - critical
    {"/var/log", 95.0, false},   // Log directory - warning only
    {"/tmp", 85.0, false},       // Temp directory - warning only
}

for i, mp := range mountPoints {
    check := DiskSpaceHealthCheckCrossPlatform(mp.path, mp.threshold)
    check.Name = fmt.Sprintf("disk_space_%d", i)
    check.Critical = mp.critical
    healthManager.RegisterCheck(check)
}
```

## Dependencies

Add to your `go.mod`:

```go
// For cross-platform implementation (recommended)
require github.com/shirou/gopsutil/v3 v3.23.0

// For platform-specific implementations
require golang.org/x/sys v0.0.0-20220908164124-27713097b956
```

## Best Practices

1. **Choose Appropriate Thresholds**: Different filesystems may have different optimal thresholds
2. **Monitor Multiple Mount Points**: Don't just monitor root - check data directories too
3. **Set Criticality Appropriately**: Only mark system-critical filesystems as critical
4. **Include Metadata**: Rich metadata helps with debugging and alerting
5. **Consider Cleanup Actions**: Implement automated cleanup when thresholds are exceeded

## Production Considerations

- **Alerting**: Configure alerts before disk space becomes critical
- **Automated Cleanup**: Implement log rotation and temporary file cleanup
- **Monitoring Frequency**: Balance between responsiveness and system load
- **Historical Tracking**: Store disk usage metrics for trend analysis