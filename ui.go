package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) setupUI() {
	a.app = tview.NewApplication()
	selectionColor := tcell.NewRGBColor(106, 159, 181)

	// Tab bar
	a.tabBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Options list
	a.optionsList = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(selectionColor).
		SetSelectedTextColor(tcell.ColorWhite)
	a.optionsList.SetBorder(true).
		SetTitle(" [1] Options ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDefault)
	// NOTE: Don't use ChangedFunc for refreshing addons/preview — tview may
	// fire it before GetCurrentItem() reflects the new index. Instead,
	// cursorDown/cursorUp and refreshAll handle refreshes explicitly.

	// Addons list (persistent panel)
	a.addonsList = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(selectionColor).
		SetSelectedTextColor(tcell.ColorWhite)
	a.addonsList.SetBorder(true).
		SetTitle(" [2] Overrides ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDefault)
	a.addonsList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.updatePreview()
	})

	// Preview panel
	a.previewView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	a.previewView.SetBorder(true).
		SetTitle(" Preview ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.ColorDefault)

	// Status bar
	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Navigable panels
	a.panels = []tview.Primitive{a.optionsList, a.addonsList}

	// Layout
	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.tabBar, 1, 0, false).
		AddItem(a.optionsList, 0, 2, true).
		AddItem(a.addonsList, 0, 1, false)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftFlex, 0, 1, true).
		AddItem(a.previewView, 0, 2, false)

	rootFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainFlex, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false)

	a.setupKeybindings()
	a.app.SetFocus(a.optionsList)
	a.updateBorderColors()

	a.pages = tview.NewPages().
		AddPage("main", rootFlex, true, true)
	a.app.SetRoot(a.pages, true)
}

// --- Refresh ---

func (a *App) refreshAll() {
	a.refreshOptionsList()
	a.refreshAddonsList()
	a.updateTabBar()
	a.updatePanelTitles()
	a.updatePreview()
	a.updateStatusBar()
	a.updateBorderColors()
}

func (a *App) refreshOptionsList() {
	currentIdx := a.optionsList.GetCurrentItem()
	a.optionsList.Clear()

	options := a.getCurrentOptions()
	for _, opt := range options {
		label := formatOptionLabel(opt)
		a.optionsList.AddItem(label, "", 0, nil)
	}

	if currentIdx >= len(options) {
		currentIdx = len(options) - 1
	}
	if currentIdx >= 0 {
		a.optionsList.SetCurrentItem(currentIdx)
	}
}

func (a *App) refreshAddonsList() {
	currentIdx := a.addonsList.GetCurrentItem()
	a.addonsList.Clear()

	opt := a.getSelectedOption()
	if opt == nil || len(opt.Addons) == 0 {
		return
	}

	for _, addon := range opt.Addons {
		var marker string
		if opt.ActiveAddons[addon.Name] {
			marker = "[green]\u2713[-] "
		} else {
			marker = "[darkgray]\u2717[-] "
		}
		label := fmt.Sprintf("%s[%s]%s[-] %s", marker, addon.Color, addon.Label, addon.Name)
		a.addonsList.AddItem(label, "", 0, nil)
	}

	if currentIdx >= len(opt.Addons) {
		currentIdx = len(opt.Addons) - 1
	}
	if currentIdx >= 0 {
		a.addonsList.SetCurrentItem(currentIdx)
	}
}

func formatOptionLabel(opt *Option) string {
	var b strings.Builder

	if opt.Enabled {
		b.WriteString("[green]\u25cf[-] ")
	} else {
		b.WriteString("[darkgray]\u25cb[-] ")
	}

	for _, addon := range opt.Addons {
		if opt.ActiveAddons[addon.Name] {
			b.WriteString(fmt.Sprintf("[%s]%s[-] ", addon.Color, addon.Label))
		} else {
			b.WriteString(fmt.Sprintf("[darkgray]%s[-] ", addon.Label))
		}
	}

	if opt.Enabled {
		b.WriteString(opt.Name)
	} else {
		b.WriteString(fmt.Sprintf("[darkgray]%s[-]", opt.Name))
	}

	return b.String()
}

// --- Data helpers ---

func (a *App) getCurrentOptions() []*Option {
	if a.activeTabIdx >= 0 && a.activeTabIdx < len(a.categories) {
		cat := a.categories[a.activeTabIdx]
		return a.options[cat.Name]
	}
	return nil
}

func (a *App) getSelectedOption() *Option {
	options := a.getCurrentOptions()
	idx := a.optionsList.GetCurrentItem()
	if idx >= 0 && idx < len(options) {
		return options[idx]
	}
	return nil
}

func (a *App) getSelectedAddon() *Addon {
	opt := a.getSelectedOption()
	if opt == nil || len(opt.Addons) == 0 {
		return nil
	}
	idx := a.addonsList.GetCurrentItem()
	if idx >= 0 && idx < len(opt.Addons) {
		return &opt.Addons[idx]
	}
	return nil
}

// --- Tab bar ---

func (a *App) updateTabBar() {
	var parts []string
	for i, cat := range a.categories {
		name := strings.ToUpper(cat.Name[:1]) + cat.Name[1:]
		if i == a.activeTabIdx {
			parts = append(parts, fmt.Sprintf("[green::b] %s [-:-:-]", name))
		} else {
			parts = append(parts, fmt.Sprintf("[darkgray] %s [-]", name))
		}
	}
	a.tabBar.SetText(strings.Join(parts, "\u2502"))
}

func (a *App) updatePanelTitles() {
	if a.activeTabIdx < len(a.categories) {
		catName := strings.ToUpper(a.categories[a.activeTabIdx].Name[:1]) + a.categories[a.activeTabIdx].Name[1:]
		a.optionsList.SetTitle(fmt.Sprintf(" [1] %s ", catName))
	}

	opt := a.getSelectedOption()
	if opt != nil {
		a.addonsList.SetTitle(fmt.Sprintf(" [2] %s ", opt.Name))
	} else {
		a.addonsList.SetTitle(" [2] Overrides ")
	}
}

// --- Preview ---

func (a *App) updatePreview() {
	a.previewView.Clear()

	opt := a.getSelectedOption()
	if opt == nil {
		a.previewView.SetTitle(" Preview ")
		a.previewView.SetText("[darkgray]No option selected[-]")
		return
	}

	// When focused on addons panel, show raw addon YAML
	if a.currentPanelIdx == 1 {
		addon := a.getSelectedAddon()
		if addon == nil {
			a.previewView.SetTitle(" Preview ")
			a.previewView.SetText("[darkgray]No addon selected[-]")
			return
		}

		a.previewView.SetTitle(fmt.Sprintf(" %s ", addon.Name))
		data, err := os.ReadFile(addon.File)
		if err != nil {
			a.previewView.SetText(fmt.Sprintf("[red]Error: %v[-]", err))
			return
		}

		highlighted := highlightCode(string(data), "yaml")
		a.previewView.SetText(highlighted)
		a.previewView.ScrollToBeginning()
		return
	}

	// Default: show resolved compose
	a.previewView.SetTitle(fmt.Sprintf(" %s ", opt.Name))

	resolved, err := resolveOption(opt)
	if err != nil {
		a.previewView.SetText(fmt.Sprintf("[red]Error: %v[-]", err))
		return
	}

	yamlStr, err := renderYAML(resolved)
	if err != nil {
		a.previewView.SetText(fmt.Sprintf("[red]Error: %v[-]", err))
		return
	}

	highlighted := highlightCode(yamlStr, "yaml")
	a.previewView.SetText(highlighted)
	a.previewView.ScrollToBeginning()
}

// --- Status bar ---

func (a *App) updateStatusBar() {
	a.statusBar.SetText(" [yellow]j/k[-] navigate  [yellow]space[-] toggle  [yellow]1/2[-] panels  [yellow]u[-] up  [yellow]s[-] down  [yellow]y[-] copy  [yellow]?[-] help  [yellow][\\[/\\]][-] tabs  [yellow]q[-] quit")
}

// --- Actions ---

func (a *App) toggleOption() {
	opt := a.getSelectedOption()
	if opt == nil {
		return
	}
	opt.Enabled = !opt.Enabled
	a.saveState()
	a.refreshAll()
}

func (a *App) toggleAddon() {
	opt := a.getSelectedOption()
	if opt == nil {
		return
	}

	idx := a.addonsList.GetCurrentItem()
	if idx < 0 || idx >= len(opt.Addons) {
		return
	}

	addonName := opt.Addons[idx].Name
	if opt.ActiveAddons[addonName] {
		delete(opt.ActiveAddons, addonName)
	} else {
		opt.ActiveAddons[addonName] = true
	}

	a.saveState()
	a.refreshAddonsList()
	a.refreshOptionsList()
	a.updatePreview()
}

// --- Confirm modals ---

func (a *App) showComposeUpConfirm() {
	a.confirmOpen = true
	a.confirmAction = func() {
		a.dockerComposeUp()
	}

	text := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow::b]Docker Compose Up[-:-:-]\n\nRun [green]docker compose up -d[-] with the current configuration?\n\n[green]Enter[-] to confirm    [yellow]Esc/q[-] to cancel")

	text.SetBorder(true).
		SetTitle(" Confirm ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddPage("confirm", modal(text, 55, 9), true, true)
	a.app.SetFocus(text)
}

func (a *App) showComposeDownConfirm() {
	a.confirmOpen = true
	a.confirmAction = func() {
		a.dockerComposeDown()
	}

	text := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow::b]Docker Compose Down[-:-:-]\n\nRun [red]docker compose down[-] to stop all services?\n\n[green]Enter[-] to confirm    [yellow]Esc/q[-] to cancel")

	text.SetBorder(true).
		SetTitle(" Confirm ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorRed)

	a.pages.AddPage("confirm", modal(text, 55, 9), true, true)
	a.app.SetFocus(text)
}

func (a *App) closeConfirm() {
	a.confirmOpen = false
	a.confirmAction = nil
	a.pages.RemovePage("confirm")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// --- Help modal ---

func (a *App) showHelp() {
	a.helpOpen = true

	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetText("[yellow::b]LazyRMSS[-:-:-]\n\n" +
			"[green]Navigation:[-]\n" +
			"  j / k         Move cursor\n" +
			"  J / K         Scroll preview\n" +
			"  [ / ]         Prev / next tab\n" +
			"  1             Jump to options\n" +
			"  2 / a         Jump to overrides\n" +
			"  Tab           Cycle panels\n" +
			"  Esc           Back / quit\n\n" +
			"[green]Actions:[-]\n" +
			"  Space / Enter Toggle item\n" +
			"  u             Docker compose up\n" +
			"  s             Docker compose down\n" +
			"  y             Copy preview YAML\n" +
			"  Y             Copy global compose\n\n" +
			"[green]Meta:[-]\n" +
			"  q             Quit\n" +
			"  ?             This help\n\n" +
			"[darkgray]Press Escape or q to close[-]")

	helpText.SetBorder(true).
		SetTitle(" Help ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddPage("help", modal(helpText, 45, 24), true, true)
	a.app.SetFocus(helpText)
}

func (a *App) closeHelp() {
	a.helpOpen = false
	a.pages.RemovePage("help")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// --- Clipboard ---

func (a *App) copyPreviewToClipboard() {
	if a.currentPanelIdx == 1 {
		addon := a.getSelectedAddon()
		if addon == nil {
			return
		}
		data, err := os.ReadFile(addon.File)
		if err != nil {
			return
		}
		copyToClipboard(string(data))
		return
	}

	opt := a.getSelectedOption()
	if opt == nil {
		return
	}
	resolved, err := resolveOption(opt)
	if err != nil {
		return
	}
	yamlStr, err := renderYAML(resolved)
	if err != nil {
		return
	}
	copyToClipboard(yamlStr)
}

func (a *App) copyGlobalComposeToClipboard() {
	global, err := a.buildGlobalCompose()
	if err != nil {
		return
	}
	yamlStr, err := renderYAML(global)
	if err != nil {
		return
	}
	copyToClipboard(yamlStr)
}

// --- Modal helper ---

func modal(content tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(content, height, 0, true).
			AddItem(nil, 0, 1, false), width, 0, true).
		AddItem(nil, 0, 1, false)
}
