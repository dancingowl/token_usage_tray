//go:build windows

package main

import "golang.org/x/sys/windows/registry"

func windowsThemeDark(value string) bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	light, _, err := key.GetIntegerValue(value)
	return err == nil && light == 0
}

func systemAppDark() bool     { return windowsThemeDark("AppsUseLightTheme") }
func systemTaskbarDark() bool { return windowsThemeDark("SystemUsesLightTheme") }
