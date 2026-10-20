package telemetry

import (
	"fmt"

	"github.com/grafana/pyroscope-go"
)

func InitProfiling(serviceName, serverAddress string) (func() error, error) {
	if serverAddress == "" {
		return func() error { return nil }, nil
	}

	pyroscopeProfiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: serviceName,
		ServerAddress:   serverAddress,
		Logger:          pyroscope.StandardLogger,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("start Pyroscope profiler: %w", err)
	}

	return pyroscopeProfiler.Stop, nil
}
