// internal/scraper/proxy_utils.go
package scraper

import (
	"fmt"
	"github.com/valpere/DataScrapexter/internal/proxy"
)

// ParseRotationStrategy converts string rotation strategy to proxy.RotationStrategy
func ParseRotationStrategy(strategy string) (proxy.RotationStrategy, error) {
	switch strategy {
	case "round_robin", "":
		return proxy.RotationRoundRobin, nil
	case "random":
		return proxy.RotationRandom, nil
	case "weighted":
		return proxy.RotationWeighted, nil
	case "healthy":
		return proxy.RotationHealthy, nil
	default:
		return "", fmt.Errorf("unsupported rotation strategy: %s", strategy)
	}
}
