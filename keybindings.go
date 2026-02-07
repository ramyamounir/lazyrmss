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
			case '2', 'a':
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
			case 'u':
				a.showComposeUpConfirm()
				return nil
			case 's':
				a.showComposeDownConfirm()
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
