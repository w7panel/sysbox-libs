package containerdUtils

import (
	"os"
	"testing"
)

func TestGetDataRoot(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		configContent string
		expectedRoot  string
		expectError   bool
	}{
		{
			name:       "Config with root entry",
			configPath: "/etc/containerd/containerd.toml",
			configContent: `
version = 2

root = "/var/lib/desktop-containerd/daemon"
state = "/run/containerd"

oom_score = 0
imports = ["/etc/containerd/runtime_*.toml", "./debug.toml"]

[grpc]
  address = "/run/containerd/containerd.sock"
  uid = 0
  gid = 0

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "k8s.gcr.io/pause:3.2"
  [plugins."io.containerd.snapshotter.v1.overlayfs"]
    root_path = "/var/lib/containerd/snapshotter"
`,
			expectedRoot: "/var/lib/desktop-containerd/daemon",
			expectError:  false,
		},
		{
			name:       "Config without root entry",
			configPath: "/etc/containerd/config.toml",
			configContent: `
version = 2

state = "/run/containerd"
oom_score = 0
imports = ["/etc/containerd/runtime_*.toml", "./debug.toml"]

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "k8s.gcr.io/pause:3.2"
  [plugins."io.containerd.snapshotter.v1.overlayfs"]
    root_path = "/var/lib/containerd/snapshotter"
`,
			expectedRoot: "/var/lib/containerd", // Default path
			expectError:  false,
		},
		{
			name:         "Nonexistent config file",
			configPath:   "/path/to/nowhere",
			expectedRoot: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := "/nonexistent/config.toml"
			if tt.configContent != "" {
				configPath = writeContainerdConfig(t, tt.configContent)
			}

			root, err := parseDataRoot(configPath)

			// Check if an error was expected or not
			if tt.expectError && err == nil {
				t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
			}

			// Check the expected root path if no error was expected
			if !tt.expectError && root != tt.expectedRoot {
				t.Fatalf("Expected root: %s, got: %s", tt.expectedRoot, root)
			}
		})
	}
}

func TestParseSandboxImage_returnsConfiguredImage_whenGrpcCRIConfigSetsSandboxImage(t *testing.T) {
	// Given
	configPath := writeContainerdConfig(t, `
version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "registry.example/pause:9.9"
`)

	// When
	image, err := parseSandboxImage(configPath)

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if image != "registry.example/pause:9.9" {
		t.Fatalf("Expected sandbox image: %s, got: %s", "registry.example/pause:9.9", image)
	}
}

func TestParseSandboxImage_returnsConfiguredImage_whenK3sCRIImagesConfigSetsSandboxImage(t *testing.T) {
	// Given
	configPath := writeContainerdConfig(t, `
version = 3

[plugins]
  [plugins."io.containerd.cri.v1.images"]
    sandbox_image = "registry.example/k3s-pause:9.9"
`)

	// When
	image, err := parseSandboxImage(configPath)

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if image != "registry.example/k3s-pause:9.9" {
		t.Fatalf("Expected sandbox image: %s, got: %s", "registry.example/k3s-pause:9.9", image)
	}
}

func TestParseSandboxImage_returnsDefaultImage_whenConfigOmitsSandboxImage(t *testing.T) {
	// Given
	configPath := writeContainerdConfig(t, `
version = 3

[plugins]
  [plugins."io.containerd.snapshotter.v1.overlayfs"]
    root_path = "/var/lib/containerd/snapshotter"
`)

	// When
	image, err := parseSandboxImage(configPath)

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if image != defaultSandboxImage {
		t.Fatalf("Expected sandbox image: %s, got: %s", defaultSandboxImage, image)
	}
}

func TestGetSandboxImageFromDir_returnsConfiguredImage(t *testing.T) {
	// Given
	configDir := writeContainerdConfigDir(t, `
version = 3

[plugins]
  [plugins."io.containerd.cri.v1.images"]
    sandbox_image = "registry.example/k3s-pause:9.9"
`)

	// When
	image, err := GetSandboxImageFromDir(configDir)

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if image != "registry.example/k3s-pause:9.9" {
		t.Fatalf("Expected sandbox image: %s, got: %s", "registry.example/k3s-pause:9.9", image)
	}
}

func TestGetSandboxImageFromDir_returnsPinnedSandboxImage(t *testing.T) {
	// Given
	configDir := writeContainerdConfigDir(t, `
version = 3

[plugins]
  [plugins."io.containerd.cri.v1.images".pinned_images]
    sandbox = "registry.example/pinned-pause:3.10"
`)

	// When
	image, err := GetSandboxImageFromDir(configDir)

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if image != "registry.example/pinned-pause:3.10" {
		t.Fatalf("Expected sandbox image: %s, got: %s", "registry.example/pinned-pause:3.10", image)
	}
}

func TestGetGRPCAddress_returnsConfiguredAddress(t *testing.T) {
	// Given
	configPath := writeContainerdConfig(t, `
version = 3

[grpc]
  address = "/run/k3s/containerd/containerd.sock"
`)
	withConfigPath(t, []string{configPath})

	// When
	address, err := GetGRPCAddress()

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if address != "/run/k3s/containerd/containerd.sock" {
		t.Fatalf("Expected configured grpc address, got: %s", address)
	}
}

func TestGetGRPCAddress_returnsDefaultAddress_whenConfigOmitsAddress(t *testing.T) {
	// Given
	configPath := writeContainerdConfig(t, `
version = 3
`)
	withConfigPath(t, []string{configPath})

	// When
	address, err := GetGRPCAddress()

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if address != defaultGRPCAddress {
		t.Fatalf("Expected default grpc address, got: %s", address)
	}
}

func TestGetGRPCAddress_returnsDefaultAddress_whenConfigFilesAreMissing(t *testing.T) {
	// Given
	withConfigPath(t, []string{"/path/to/missing-containerd-config.toml"})

	// When
	address, err := GetGRPCAddress()

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if address != defaultGRPCAddress {
		t.Fatalf("Expected default grpc address, got: %s", address)
	}
}

func TestGetProxyPluginCapabilities_returnsConfiguredCapabilities(t *testing.T) {
	// Given
	configPath := writeContainerdConfig(t, `
version = 3

[proxy_plugins."sysbox"]
  type = "snapshot"
  address = "/run/sysbox-snapshotter.sock"
  capabilities = ["remap-ids", "walk-diff"]
`)
	withConfigPath(t, []string{configPath})

	// When
	capabilities, err := GetProxyPluginCapabilities("sysbox")

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(capabilities) != 2 || capabilities[0] != "remap-ids" || capabilities[1] != "walk-diff" {
		t.Fatalf("Expected configured capabilities, got: %#v", capabilities)
	}
}

func TestGetProxyPluginCapabilities_returnsEmpty_whenCapabilitiesAreOmitted(t *testing.T) {
	// Given
	configPath := writeContainerdConfig(t, `
version = 3

[proxy_plugins."sysbox"]
  type = "snapshot"
  address = "/run/sysbox-snapshotter.sock"
`)
	withConfigPath(t, []string{configPath})

	// When
	capabilities, err := GetProxyPluginCapabilities("sysbox")

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(capabilities) != 0 {
		t.Fatalf("Expected no capabilities, got: %#v", capabilities)
	}
}

func TestGetProxyPluginCapabilities_returnsEmpty_whenPluginIsMissing(t *testing.T) {
	// Given
	configPath := writeContainerdConfig(t, `
version = 3

[proxy_plugins."other"]
  type = "snapshot"
  address = "/run/other.sock"
  capabilities = ["remap-ids"]
`)
	withConfigPath(t, []string{configPath})

	// When
	capabilities, err := GetProxyPluginCapabilities("sysbox")

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(capabilities) != 0 {
		t.Fatalf("Expected no capabilities, got: %#v", capabilities)
	}
}

func TestGetProxyPluginCapabilities_returnsEmpty_whenConfigFilesAreMissing(t *testing.T) {
	// Given
	withConfigPath(t, []string{"/path/to/missing-containerd-config.toml"})

	// When
	capabilities, err := GetProxyPluginCapabilities("sysbox")

	// Then
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(capabilities) != 0 {
		t.Fatalf("Expected no capabilities, got: %#v", capabilities)
	}
}

func withConfigPath(t *testing.T, paths []string) {
	t.Helper()
	original := configPath
	configPath = paths
	t.Cleanup(func() { configPath = original })
}

func writeContainerdConfig(t *testing.T, configContent string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	if _, err = tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err = tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	return tmpFile.Name()
}

func writeContainerdConfigDir(t *testing.T, configContent string) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "containerd-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	if err = os.WriteFile(tmpDir+"/config.toml", []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config.toml: %v", err)
	}
	return tmpDir
}
