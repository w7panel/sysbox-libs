// Copyright: (C) 2024 Nestybox Inc.  All rights reserved.
package containerdUtils

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Location of containerd config file
// (see https://github.com/containerd/containerd/blob/main/docs/man/containerd-config.toml.5.md)
var (
	configPath = []string{
		"/etc/containerd/containerd.toml",
		"/etc/containerd/config.toml",
		"/usr/local/etc/containerd/config.toml",
	}

	defaultDataRoot     = "/var/lib/containerd"
	defaultSandboxImage = "rancher/mirrored-pause:3.6"
)

type containerdConfig struct {
	Root    string                            `toml:"Root"`
	Plugins map[string]containerdPluginConfig `toml:"plugins"`
}

type containerdPluginConfig struct {
	SandboxImage string `toml:"sandbox_image"`
	Images       struct {
		SandboxImage string `toml:"sandbox_image"`
	} `toml:"images"`
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
	}
	return defaultSandboxImage, nil
}

func parseConfig(path string, config *containerdConfig) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := toml.NewDecoder(f).Decode(config); err != nil {
		return fmt.Errorf("could not decode %s: %w", path, err)
	}
	return nil
}
