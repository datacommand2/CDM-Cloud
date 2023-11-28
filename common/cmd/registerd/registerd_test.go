//go:build kubernetes_registry
// +build kubernetes_registry

package main

import (
	"github.com/stretchr/testify/assert"
	"go-micro.dev/v4/registry"

	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
}

func TestRegister(t *testing.T) {
	defer func() {
		OsExiter = os.Exit
	}()

	for _, tc := range []struct {
		Name          string
		Version       string
		AdvertisePort string
		Metadata      string
		Solution      string
		Description   string
		Registry      string
		Success       bool
	}{
		{
			// normal
			Name:          "service_test",
			Version:       "1.0.0",
			AdvertisePort: "8000",
			Metadata:      "a=1,b=2,c", // c is ignored
			Solution:      "CDM-Cloud",
			Description:   "register service",
			Registry:      "kubernetes",
			Success:       true,
		},
		{
			// empty name
			Version:       "unknown",
			AdvertisePort: "8000",
			Solution:      "CDM-Cloud",
			Description:   "register service",
			Success:       false,
		},
		{
			// empty version
			Name:          "service_test",
			AdvertisePort: "8000",
			Solution:      "CDM-Cloud",
			Description:   "register service",
			Success:       false,
		},
		{
			// empty advertise port
			Name:        "service_test",
			Version:     "1.0.0",
			Solution:    "CDM-Cloud",
			Description: "register service",
			Success:     false,
		},
		{
			// empty solution
			Name:          "service_test",
			Version:       "1.0.0",
			AdvertisePort: "8000",
			Description:   "register service",
			Success:       false,
		},
		{
			// empty description
			Name:          "service_test",
			Version:       "1.0.0",
			AdvertisePort: "8000",
			Solution:      "CDM-Cloud",
			Success:       false,
		},
	} {
		ch := make(chan struct{})
		OsExiter = func(c int) {
			close(ch)
		}

		// set abnormal test environments
		_ = os.Unsetenv("CDM_SERVICE_NAME")
		_ = os.Unsetenv("CDM_SERVICE_VERSION")
		_ = os.Unsetenv("CDM_SERVICE_ADVERTISE_PORT")
		_ = os.Unsetenv("CDM_SERVICE_METADATA")
		_ = os.Unsetenv("CDM_SOLUTION_NAME")
		_ = os.Unsetenv("CDM_SERVICE_DESCRIPTION")

		if tc.Name != "" {
			_ = os.Setenv("CDM_SERVICE_NAME", tc.Name)
		}
		if tc.Version != "" {
			_ = os.Setenv("CDM_SERVICE_VERSION", tc.Version)
		}
		if tc.AdvertisePort != "" {
			_ = os.Setenv("CDM_SERVICE_ADVERTISE_PORT", tc.AdvertisePort)
		}
		if tc.Metadata != "" {
			_ = os.Setenv("CDM_SERVICE_METADATA", tc.Metadata)
		}
		if tc.Solution != "" {
			_ = os.Setenv("CDM_SOLUTION_NAME", tc.Solution)
		}
		if tc.Description != "" {
			_ = os.Setenv("CDM_SERVICE_DESCRIPTION", tc.Description)
		}
		if tc.Registry != "" {
			_ = os.Setenv("MICRO_REGISTRY", tc.Registry)
		}

		// execute
		go main()

		select {
		case <-ch:
			if tc.Success {
				t.Error("Register success is expected but it is failed.")
			}
			continue

		case <-time.After(5 * time.Second):
			if !tc.Success {
				t.Error("Register fail is expected but it is succeed.")
			}
		}

		// check if registered
		s, err := registry.GetService(tc.Name)
		assert.NoError(t, err)
		assert.NotNil(t, s)
		assert.NotEmpty(t, s)

		assert.Equal(t, tc.Name, s[0].Name)
		assert.Equal(t, tc.Version, s[0].Version)
		assert.Contains(t, s[0].Nodes[0].Address, tc.AdvertisePort)
		assert.Contains(t, s[0].Nodes[0].Metadata, "a")
		assert.Equal(t, s[0].Nodes[0].Metadata["a"], "1")
		assert.Contains(t, s[0].Nodes[0].Metadata, "b")
		assert.Equal(t, s[0].Nodes[0].Metadata["b"], "2")
		assert.NotContains(t, s[0].Nodes[0].Metadata, "c")
		assert.Contains(t, s[0].Nodes[0].Metadata, "CDM_SOLUTION_NAME")
		assert.Equal(t, s[0].Nodes[0].Metadata["CDM_SOLUTION_NAME"], tc.Solution)
		assert.Contains(t, s[0].Nodes[0].Metadata, "CDM_SERVICE_DESCRIPTION")
		assert.Equal(t, s[0].Nodes[0].Metadata["CDM_SERVICE_DESCRIPTION"], tc.Description)

		for _, ss := range s {
			_ = registry.Deregister(ss)
		}
	}
}
