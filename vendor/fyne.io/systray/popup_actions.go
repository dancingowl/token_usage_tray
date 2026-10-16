package systray

// ClickActionCh reports quick actions from the Windows click summary. The
// native window never blocks when the receiver is busy. Other platforms do not
// send actions on this channel.
var ClickActionCh = make(chan string, 8)

func notifyClickAction(action string) {
	select {
	case ClickActionCh <- action:
	default:
	}
}
