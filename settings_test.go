package main

import (
	"os"
	"testing"
)

func TestTraySettingsDefaultAndRoundTrip(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	settings, err := loadTraySettings()
	if err != nil || settings.IconStyle != iconStyleDonut {
		t.Fatalf("missing settings: %+v, %v", settings, err)
	}
	for _, style := range []iconStyle{iconStyleNumber, iconStyleDonut} {
		if err := saveTraySettings(traySettings{IconStyle: style}); err != nil {
			t.Fatal(err)
		}
		loaded, err := loadTraySettings()
		if err != nil || loaded.IconStyle != style {
			t.Fatalf("saved %q, loaded %+v: %v", style, loaded, err)
		}
	}
}

func TestTraySettingsRejectsInvalidWithoutOverwriting(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	if err := saveTraySettings(traySettings{IconStyle: iconStyleNumber}); err != nil {
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
