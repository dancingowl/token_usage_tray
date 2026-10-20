package main

import (
	"errors"
	"net"
	"time"

	"fyne.io/systray"
)

var runtimeAutoDiscover bool
var runtimeHandingOff bool
var runtimeCloseListener func() error

// Once a sibling has been observed, honor the user's later Quit action.
var runtimeObservedSiblings = make(map[string]bool)

func discoverAgents() enabledAgents {
	c, x := detectAgents()
	return enabledAgents{c, x, detectAntigravity()}
}

func filterDiscoveredAgents(found enabledAgents, owner string) enabledAgents {
	switch owner {
	case "claude":
		return enabledAgents{claude: found.claude}
	case "codex":
		return enabledAgents{codex: found.codex}
	case "antigravity":
		return enabledAgents{antigravity: found.antigravity}
	}
	return found
}

func agentNames(found enabledAgents) []string {
	var names []string
	if found.claude {
		names = append(names, "claude")
	}
	if found.codex {
		names = append(names, "codex")
	}
	if found.antigravity {
		names = append(names, "antigravity")
	}
	return names
}

func needsAgentSplit(found enabledAgents, owner string, split bool) bool {
	return split && owner == "" && len(agentNames(found)) > 1
}

// Release the shared default port before starting its future Claude owner.
// Callbacks keep the transition order verifiable without operating a real tray.
func handoffAgentProcesses(found enabledAgents, closeListener func() error, spawn func([]string), quit func()) error {
	if err := closeListener(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	spawn(agentNames(found))
	quit()
	return nil
}

func agentPortInUse(name string) bool {
	c, err := net.DialTimeout("tcp", ports[name], 150*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// Called only by the tray event loop; collector goroutines receive a copied selection.
func discoverRuntimeAgents() bool {
	if runtimeHandingOff {
		return false
	}
	found := discoverAgents()
	if needsAgentSplit(found, tag(), splitPerAgent()) {
		err := handoffAgentProcesses(found, func() error {
			err := runtimeCloseListener()
			if err == nil || errors.Is(err, net.ErrClosed) {
				runtimeHandingOff = true
			}
			return err
		}, spawnChildren, systray.Quit)
		if err != nil {
			logf("agent handoff: %v", err)
		}
		return false
	}
	occupied := func(name string) bool {
		if runtimeObservedSiblings[name] {
			return true
		}
		if agentPortInUse(name) {
			runtimeObservedSiblings[name] = true
			return true
		}
		return false
	}
	return reconnectAgents(found, tag(), runtimeAutoDiscover, splitPerAgent(), occupied, spawnChildren)
}

func reconnectAgents(found enabledAgents, owner string, autoDiscover, split bool, occupied func(string) bool, spawn func([]string)) bool {
	if split && autoDiscover && owner != "" {
		var missing []string
		for _, name := range agentNames(found) {
			if name != owner && !occupied(name) {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			spawn(missing)
		}
	}
	selected := filterDiscoveredAgents(found, owner)
	next := enabledAgents{agents.claude || selected.claude, agents.codex || selected.codex, agents.antigravity || selected.antigravity}
	changed := next != agents
	agents = next
	return changed
}
