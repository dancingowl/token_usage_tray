package main

import (
	"os"
	"testing"
)

func TestTraySettingsDefaultAndRoundTrip(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	settings, err := loadTraySettings()
	if err != nil || settings.IconStyle != iconStyleDonut || settings.Theme != "system" || settings.AlertsPausedUntil != 0 {
		t.Fatalf("missing settings: %+v, %v", settings, err)
	}
	for _, style := range []iconStyle{iconStyleNumber, iconStyleDonut} {
		next := defaultTraySettings()
		next.IconStyle = style
		if err := saveTraySettings(next); err != nil {
			t.Fatal(err)
		}
		loaded, err := loadTraySettings()
		if err != nil || loaded.IconStyle != style {
			t.Fatalf("saved %q, loaded %+v: %v", style, loaded, err)
		}
	}
}

func TestThemeAndPauseSettingsValidation(t *testing.T) {
	for _, theme := range []string{"system", "light", "dark", ""} {
		s := defaultTraySettings()
		s.Theme = theme
		s.AlertsPausedUntil = 2000000000
		if err := validateTraySettings(s); err != nil {
			t.Fatalf("valid theme/pause: %v", err)
		}
	}
	s := defaultTraySettings()
	s.Theme = "unsupported"
	if validateTraySettings(s) == nil {
		t.Fatal("invalid theme accepted")
	}
	s = defaultTraySettings()
	s.AlertsPausedUntil = -1
	if validateTraySettings(s) == nil {
		t.Fatal("negative pause timestamp accepted")
	}
}

func TestTraySettingsRejectsInvalidWithoutOverwriting(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	next := defaultTraySettings()
	next.IconStyle = iconStyleNumber
	if err := saveTraySettings(next); err != nil {
		t.Fatal(err)
	}
	if err := saveTraySettings(traySettings{IconStyle: "unsupported"}); err == nil {
		t.Fatal("unsupported style accepted")
	}
	settings, err := loadTraySettings()
	if err != nil || settings.IconStyle != iconStyleNumber {
		t.Fatalf("valid preference lost: %+v, %v", settings, err)
	}
}

func TestTraySettingsMalformedFileFallsBack(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	for _, content := range []string{`{"icon_style":"unsupported"}`, `{"icon_style":`, `{"icon_style":12}`} {
		if err := os.WriteFile(settingsPath(), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		settings, err := loadTraySettings()
		if err == nil || settings.IconStyle != iconStyleDonut {
			t.Fatalf("invalid settings %q: %+v, %v", content, settings, err)
		}
	}
	if err := os.WriteFile(settingsPath(), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := loadTraySettings()
	if err != nil || settings.IconStyle != iconStyleDonut {
		t.Fatalf("default field: %+v, %v", settings, err)
	}
	if err := os.WriteFile(settingsPath(), []byte(`{"icon_style":"donut-number"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err = loadTraySettings()
	if err != nil || settings.IconStyle != iconStyleDonut {
		t.Fatalf("removed style fallback: %+v, %v", settings, err)
	}
}
