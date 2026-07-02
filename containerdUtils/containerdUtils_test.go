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
