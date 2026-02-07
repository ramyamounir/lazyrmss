package main

import "github.com/gdamore/tcell/v2"

func (a *App) setupKeybindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		// === MODAL PRIORITY CHAIN ===

		if a.helpOpen {
			if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
				a.closeHelp()
				return nil
			}
			return event
		}

		if a.confirmOpen {
			if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
				a.closeConfirm()
				return nil
			}
			if event.Key() == tcell.KeyEnter {
				if a.confirmAction != nil {
					a.confirmAction()
				}
				a.closeConfirm()
				return nil
			}
			return event
		}

		// === MAIN KEYBINDINGS ===

		// Panel 0 only: Docker compose actions
		if event.Key() == tcell.KeyRune && a.currentPanelIdx == 0 {
			switch event.Rune() {
			case 'U':
				a.confirmGlobalAction("Up All", "up -d", tcell.ColorGreen, "up", "-d")
				return nil
			case 'D':
				a.confirmGlobalAction("Down All", "down", tcell.ColorRed, "down")
				return nil
			case 's':
				a.confirmSingleAction("Stop", "stop", tcell.ColorYellow, func() { a.dockerDirectSingle("stop") })
				return nil
			case 'S':
				a.confirmGlobalAction("Stop All", "stop", tcell.ColorYellow, "stop")
				return nil
			case 'c':
				a.confirmSingleAction("Start", "start", tcell.ColorGreen, func() { a.dockerDirectSingle("start") })
				return nil
			case 'C':
				a.confirmGlobalAction("Start All", "start", tcell.ColorGreen, "start")
				return nil
			case 'r':
				a.confirmSingleAction("Restart", "restart", tcell.ColorYellow, func() { a.dockerDirectSingle("restart") })
				return nil
			case 'R':
				a.confirmGlobalAction("Restart All", "restart", tcell.ColorYellow, "restart")
				return nil
			case 'p':
				a.confirmSingleAction("Pull", "pull", tcell.ColorBlue, func() { a.dockerPullSingle() })
				return nil
			case 'P':
				a.confirmGlobalAction("Pull All", "pull", tcell.ColorBlue, "pull")
				return nil
			}
		}

		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				a.app.Stop()
				return nil
			case '1':
				a.focusPanel(0)
				a.updatePreview()
				return nil
			case '2':
				a.focusPanel(1)
				a.updatePreview()
				return nil
			case 'h':
				a.prevPanel()
				a.updatePreview()
				return nil
			case 'l':
				a.nextPanel()
				a.updatePreview()
				return nil
			case 'j':
				a.cursorDown()
				return nil
			case 'k':
				a.cursorUp()
				return nil
			case 'J':
				a.scrollPreviewDown()
				return nil
			case 'K':
				a.scrollPreviewUp()
				return nil
			case '[':
				a.prevTab()
				return nil
			case ']':
				a.nextTab()
				return nil
			case ' ':
				if a.currentPanelIdx == 0 {
					a.toggleOption()
				} else if a.currentPanelIdx == 1 {
					a.toggleAddon()
				}
				return nil
			case 'e':
				a.editResourceFile()
				return nil
			case 'y':
				a.copyPreviewToClipboard()
				return nil
			case 'Y':
				a.copyGlobalComposeToClipboard()
				return nil
			case '?':
				a.showHelp()
				return nil
			}
		case tcell.KeyTab:
			a.nextPanel()
			a.updatePreview()
			return nil
		case tcell.KeyBacktab:
			a.prevPanel()
			a.updatePreview()
			return nil
		case tcell.KeyEnter:
			if a.currentPanelIdx == 0 {
				a.toggleOption()
			} else if a.currentPanelIdx == 1 {
				a.toggleAddon()
			}
			return nil
		case tcell.KeyEsc:
			if a.currentPanelIdx == 1 {
				a.focusPanel(0)
				a.updatePreview()
				return nil
			}
			a.app.Stop()
			return nil
		}

		return event
	})
}
