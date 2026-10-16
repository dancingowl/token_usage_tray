//go:build windows

package systray

import "log"

const (
	wmTrayVisibility      = 0x8000 + 40 // WM_APP; dispatched on the tray OS thread.
	wmClickSummaryChanged = 0x8000 + 41
)

// SetVisible hides or restores the notification icon without stopping the
// event loop. Icon and tooltip updates made while hidden are used on restore.
// Calls before initialization also preserve the requested visibility.
func SetVisible(visible bool) {
	wt.setVisible(visible)
}

func (t *winTray) setVisible(visible bool) {
	t.muNID.Lock()
	t.hidden = !visible
	t.muNID.Unlock()
	if t.isReady() {
		if ok, _, err := pPostMessage.Call(uintptr(t.window), wmTrayVisibility, 0, 0); ok == 0 {
			log.Printf("systray error: unable to queue visibility: %s", err)
		}
	}
}

// Only the tray window's thread calls this; muNID serializes Shell calls with
// concurrent icon and hover-text updates.
func (t *winTray) applyVisibility() error {
	t.muNID.Lock()
	defer t.muNID.Unlock()
	visible := !t.hidden
	if t.nid == nil || t.iconAdded == visible {
		return nil
	}
	if t.hidden {
		if err := t.nid.delete(); err != nil {
			return err
		}
		t.iconAdded = false
		return nil
	}
	if err := t.nid.add(); err != nil {
		return err
	}
	t.iconAdded = true
	return nil
}
