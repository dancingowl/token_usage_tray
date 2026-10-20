package main

import (
	"os"
	"sync"
)

var settingsMutationMu sync.Mutex

// Windows agent processes share settings.json. Lock a separate, stable file:
// the settings file itself is replaced atomically while the lock is held.
func acquireSettingsMutation() (func(), error) {
	settingsMutationMu.Lock()
	f, err := os.OpenFile(settingsPath()+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		settingsMutationMu.Unlock()
		return nil, err
	}
	if err := lockSettingsHandle(f); err != nil {
		_ = f.Close()
		settingsMutationMu.Unlock()
		return nil, err
	}
	return func() {
		unlockSettingsHandle(f)
		_ = f.Close()
		settingsMutationMu.Unlock()
	}, nil
}

func updateTraySettings(change func(*traySettings) error) (traySettings, error) {
	unlock, err := acquireSettingsMutation()
	if err != nil {
		return traySettings{}, err
	}
	defer unlock()
	settings, err := loadTraySettings()
	if err != nil {
		return settings, err
	}
	if err := change(&settings); err != nil {
		return settings, err
	}
	return settings, saveTraySettings(settings)
}
