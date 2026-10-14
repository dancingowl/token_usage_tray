package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// All agent processes share one choice. State and usage caches remain separate.
type traySettings struct {
	IconStyle iconStyle `json:"icon_style"`
}

func defaultTraySettings() traySettings { return traySettings{IconStyle: iconStyleDonut} }

func validIconStyle(style iconStyle) bool {
	return style == iconStyleDonut || style == iconStyleNumber
}

func settingsPath() string { return filepath.Join(dataDir(), "settings.json") }

func loadTraySettings() (traySettings, error) {
	settings := defaultTraySettings()
	b, err := os.ReadFile(settingsPath())
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if err := json.Unmarshal(b, &settings); err != nil {
		return defaultTraySettings(), fmt.Errorf("아이콘 설정을 읽지 못했다: %w", err)
	}
	// 폐기된 도넛+숫자 설정은 기본 도넛으로 읽는다.
	if settings.IconStyle == "donut-number" {
		return defaultTraySettings(), nil
	}
	if !validIconStyle(settings.IconStyle) {
		return defaultTraySettings(), fmt.Errorf("지원하지 않는 아이콘 모양: %q", settings.IconStyle)
	}
	return settings, nil
}

func saveTraySettings(settings traySettings) error {
	if !validIconStyle(settings.IconStyle) {
		return fmt.Errorf("지원하지 않는 아이콘 모양: %q", settings.IconStyle)
	}
	b, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	path := settingsPath()
	f, err := os.CreateTemp(filepath.Dir(path), ".settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
