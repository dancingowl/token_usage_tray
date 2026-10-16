package main

func hasAvailableAgent(s State) bool {
	return s.Claude.Available || s.Codex.Available || s.Antigravity.Available
}

func trayShouldBeVisible(own, all State, settings traySettings, ownTag string) bool {
	if hasAvailableAgent(own) {
		return true
	}
	if hasAvailableAgent(trayVisibleState(all, settings)) {
		return false
	}
	// Keep one settings entry accessible even when every connected agent is hidden.
	if ownTag == "" {
		return true
	}
	if all.Claude.Available {
		return ownTag == "claude"
	}
	if all.Codex.Available {
		return ownTag == "codex"
	}
	if all.Antigravity.Available {
		return ownTag == "antigravity"
	}
	return true
}

func displayWindowLabel(window string) string {
	switch window {
	case "short":
		return "단기 한도"
	case "urgent":
		return "가장 부족한 한도"
	default:
		return "주간 한도"
	}
}
