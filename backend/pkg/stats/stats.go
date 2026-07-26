package stats

import (
	"fmt"
	"time"

	"github.com/coder/clistat"
)

// Stats contains system resource usage metrics.
type Stats struct {
	Timestamp       string          `json:"timestamp,omitempty"`
	HostCPU         *clistat.Result `json:"host_cpu,omitempty"`
	HostMemory      *clistat.Result `json:"host_memory,omitempty"`
	ContainerCPU    *clistat.Result `json:"container_cpu,omitempty"`
	ContainerMemory *clistat.Result `json:"container_memory,omitempty"`
	Disk            *clistat.Result `json:"disk,omitempty"`
	IsContainerized bool            `json:"is_containerized,omitempty"`
}

// Read returns the current system resource usage.
func Read() (Stats, error) {
	statter, err := clistat.New()
	if err != nil {
		return Stats{}, fmt.Errorf("create stats reader: %w", err)
	}

	timestamp := time.Now().String()

	hostCPU, err := statter.HostCPU()
	if err != nil {
		return Stats{}, fmt.Errorf("read host CPU stats: %w", err)
	}

	hostMem, err := statter.HostMemory(clistat.PrefixGibi)
	if err != nil {
		return Stats{}, fmt.Errorf("read host memory stats: %w", err)
	}

	disk, err := statter.Disk(clistat.PrefixGibi, "") // Get disk usage for root path
	if err != nil {
		return Stats{}, fmt.Errorf("read disk stats: %w", err)
	}

	isContainerized, err := statter.IsContainerized()
	if err != nil {
		return Stats{}, fmt.Errorf("detect containerization: %w", err)
	}

	s := Stats{
		Timestamp:       timestamp,
		HostCPU:         hostCPU,
		HostMemory:      hostMem,
		Disk:            disk,
		IsContainerized: isContainerized,
	}

	if isContainerized {
		containerCPU, err := statter.ContainerCPU()
		if err != nil {
			return Stats{}, fmt.Errorf("read container CPU stats: %w", err)
		}

		// Get container memory if containerized
		containerMem, err := statter.ContainerMemory(clistat.PrefixGibi)
		if err != nil {
			return Stats{}, fmt.Errorf("read container memory stats: %w", err)
		}

		s.ContainerCPU = containerCPU
		s.ContainerMemory = containerMem
	}

	return s, nil
}
