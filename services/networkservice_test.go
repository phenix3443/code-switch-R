package services

import (
	"path/filepath"
	"testing"
)

func TestNetworkSettingsDefaultRelayPort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := NewNetworkService("", nil, nil, nil)

	settings, err := service.GetNetworkSettings()
	if err != nil {
		t.Fatalf("GetNetworkSettings() 返回错误: %v", err)
	}
	if settings.RelayPort != 18100 {
		t.Fatalf("期望默认 relay port 为 18100，得到 %d", settings.RelayPort)
	}
	if settings.CurrentAddress != "127.0.0.1:18100" {
		t.Fatalf("期望默认当前地址为 127.0.0.1:18100，得到 %s", settings.CurrentAddress)
	}
}

func TestNetworkSettingsPersistRelayPort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := NewNetworkService("", nil, nil, nil)

	input := NetworkSettings{
		ListenMode:     ListenModeLAN,
		RelayPort:      19100,
		WSLAutoConfig:  true,
		TargetCli:      TargetCli{ClaudeCode: true, Codex: true, Gemini: false},
		CurrentAddress: "",
	}
	if err := service.SaveNetworkSettings(input); err != nil {
		t.Fatalf("SaveNetworkSettings() 返回错误: %v", err)
	}

	settings, err := service.GetNetworkSettings()
	if err != nil {
		t.Fatalf("GetNetworkSettings() 返回错误: %v", err)
	}
	if settings.RelayPort != 19100 {
		t.Fatalf("期望持久化 relay port 为 19100，得到 %d", settings.RelayPort)
	}
	if settings.CurrentAddress != "0.0.0.0:19100" {
		t.Fatalf("期望当前地址为 0.0.0.0:19100，得到 %s", settings.CurrentAddress)
	}
}

func TestComputeListenAddressUsesRelayPort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := NewNetworkService("", nil, nil, nil)

	custom := service.computeListenAddress(NetworkSettings{
		ListenMode:    ListenModeCustom,
		CustomAddress: "10.0.0.8:29000",
		RelayPort:     19100,
	})
	if custom != "10.0.0.8:29000" {
		t.Fatalf("期望 custom 地址原样保留，得到 %s", custom)
	}

	localhost := service.computeListenAddress(NetworkSettings{
		ListenMode: ListenModeLocalhost,
		RelayPort:  19100,
	})
	if localhost != "127.0.0.1:19100" {
		t.Fatalf("期望 localhost 地址为 127.0.0.1:19100，得到 %s", localhost)
	}
}

func TestLoadRelayAddressFromNetworkSettingsUsesPersistedPort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := NewNetworkService("", nil, nil, nil)
	if err := service.SaveNetworkSettings(NetworkSettings{
		ListenMode: ListenModeLocalhost,
		RelayPort:  19200,
	}); err != nil {
		t.Fatalf("SaveNetworkSettings() 返回错误: %v", err)
	}

	addr, err := LoadRelayAddressFromNetworkSettings(filepath.Join(home, appSettingsDir, networkSettingsFile))
	if err != nil {
		t.Fatalf("LoadRelayAddressFromNetworkSettings() 返回错误: %v", err)
	}
	if addr != "127.0.0.1:19200" {
		t.Fatalf("期望读取到 127.0.0.1:19200，得到 %s", addr)
	}
}
