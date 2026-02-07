package main

import "github.com/gdamore/tcell/v2"

func (a *App) setupKeybindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		// === MODAL PRIORITY CHAIN ===

		if a.addonOpen {
			switch event.Key() {
			case tcell.KeyEsc:
				a.closeAddonPicker()
				return nil
			case tcell.KeyRune:
				switch event.Rune() {
				case 'q':
					a.closeAddonPicker()
					return nil
				case 'j':
					count := a.addonList.GetItemCount()
					current := a.addonList.GetCurrentItem()
					if current < count-1 {
						a.addonList.SetCurrentItem(current + 1)
					}
					return nil
				case 'k':
					current := a.addonList.GetCurrentItem()
					if current > 0 {
						a.addonList.SetCurrentItem(current - 1)
					}
					return nil
				case ' ':
					a.toggleAddonInPicker()
					return nil
				}
			case tcell.KeyEnter:
				a.toggleAddonInPicker()
				return nil
			}
			return event
		}

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
				return nil
			case 'h':
				a.prevPanel()
				return nil
			case 'l':
				a.nextPanel()
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
				a.toggleOption()
				return nil
			case 'a':
				a.showAddonPicker()
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
			return nil
		case tcell.KeyBacktab:
			a.prevPanel()
			return nil
		case tcell.KeyEnter:
			a.toggleOption()
			return nil
		case tcell.KeyEsc:
			a.app.Stop()
			return nil
		}

		return event
	})
}
