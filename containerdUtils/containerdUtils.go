// Copyright: (C) 2024 Nestybox Inc.  All rights reserved.
package containerdUtils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Location of containerd config file
// (see https://github.com/containerd/containerd/blob/main/docs/man/containerd-config.toml.5.md)
var (
	configPath = []string{
		"/var/lib/rancher/k3s/agent/etc/containerd/config.toml",
		"/etc/containerd/containerd.toml",
		"/etc/containerd/config.toml",
		"/usr/local/etc/containerd/config.toml",
	}

	defaultDataRoot     = "/var/lib/containerd"
	defaultSandboxImage = "rancher/mirrored-pause:3.6"
	defaultGRPCAddress  = "/run/containerd/containerd.sock"
)

type containerdConfig struct {
	Root         string                                 `toml:"Root"`
	GRPC         containerdGRPCConfig                   `toml:"grpc"`
	Plugins      map[string]containerdPluginConfig      `toml:"plugins"`
	ProxyPlugins map[string]containerdProxyPluginConfig `toml:"proxy_plugins"`
}

type containerdGRPCConfig struct {
	Address string `toml:"address"`
}

type containerdPluginConfig struct {
	SandboxImage string `toml:"sandbox_image"`
	Images       struct {
		SandboxImage string `toml:"sandbox_image"`
	} `toml:"images"`
	PinnedImages struct {
		Sandbox string `toml:"sandbox"`
	} `toml:"pinned_images"`
}

type containerdProxyPluginConfig struct {
	Capabilities []string `toml:"capabilities"`
}

// GetDataRoot returns the containerd data root directory, as read from
// the containerd config file.
func GetDataRoot() (string, error) {
	for _, path := range configPath {
		dataRoot, err := parseDataRoot(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("failed to open file %s: %w", path, err)
		}
		return dataRoot, nil
	}
	return defaultDataRoot, nil
}

func parseDataRoot(path string) (string, error) {
	var config containerdConfig
	if err := parseConfig(path, &config); err != nil {
		return "", err
	}

	// if no "root" present, assume it's the default
	if config.Root == "" {
		return defaultDataRoot, nil
	}

	return config.Root, nil
}

// GetSandboxImage returns the containerd sandbox image from the first existing
// config file, or containerd's default sandbox image when unset.
func GetSandboxImage() (string, error) {
	for _, path := range configPath {
		sandboxImage, err := parseSandboxImage(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("failed to open file %s: %w", path, err)
		}
		return sandboxImage, nil
	}
	return defaultSandboxImage, nil
}

// GetSandboxImageFromDir returns the containerd sandbox image from config.toml
// in the given directory, or containerd's default sandbox image when unset.
func GetSandboxImageFromDir(configDir string) (string, error) {
	path := filepath.Join(configDir, "config.toml")
	sandboxImage, err := parseSandboxImage(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	return sandboxImage, nil
}

func parseSandboxImage(path string) (string, error) {
	var config containerdConfig
	if err := parseConfig(path, &config); err != nil {
		return "", err
	}
	for _, pluginID := range []string{"io.containerd.grpc.v1.cri", "io.containerd.cri.v1.images"} {
		plugin, ok := config.Plugins[pluginID]
		if !ok {
			continue
		}
		if plugin.SandboxImage != "" {
			return plugin.SandboxImage, nil
		}
		if plugin.Images.SandboxImage != "" {
			return plugin.Images.SandboxImage, nil
		}
		if plugin.PinnedImages.Sandbox != "" {
			return plugin.PinnedImages.Sandbox, nil
		}
	}
	return defaultSandboxImage, nil
}

// GetGRPCAddress returns the containerd gRPC socket address.
func GetGRPCAddress() (string, error) {
	for _, path := range configPath {
		address, err := parseGRPCAddress(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("failed to open file %s: %w", path, err)
		}
		return address, nil
	}
	return defaultGRPCAddress, nil
}

func parseGRPCAddress(path string) (string, error) {
	var config containerdConfig
	if err := parseConfig(path, &config); err != nil {
		return "", err
	}
	if config.GRPC.Address == "" {
		return defaultGRPCAddress, nil
	}
	return config.GRPC.Address, nil
}

// GetProxyPluginCapabilities returns the configured capabilities for a containerd proxy plugin.
func GetProxyPluginCapabilities(pluginID string) ([]string, error) {
	for _, path := range configPath {
		capabilities, err := parseProxyPluginCapabilities(path, pluginID)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to open file %s: %w", path, err)
		}
		return capabilities, nil
	}
	return nil, nil
}

func parseProxyPluginCapabilities(path string, pluginID string) ([]string, error) {
	var config containerdConfig
	if err := parseConfig(path, &config); err != nil {
		return nil, err
	}
	plugin, ok := config.ProxyPlugins[pluginID]
	if !ok {
		return nil, nil
	}
	return plugin.Capabilities, nil
}

func parseConfig(path string, config *containerdConfig) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := toml.NewDecoder(f).Decode(config); err != nil {
		return fmt.Errorf("could not decode %s: %w", path, err)
	}
	return nil
}
