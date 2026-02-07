package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) setupUI() {
	a.app = tview.NewApplication()
	selectionColor := tcell.NewRGBColor(68, 68, 88)

	// Tab bar
	a.tabBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Options list
	a.optionsList = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedStyle(tcell.StyleDefault.Background(selectionColor))
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
		SetSelectedStyle(tcell.StyleDefault.Background(selectionColor))
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

	// Log panel
	a.logView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	a.logView.SetBorder(true).
		SetTitle(" Log ").
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

	rightFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.previewView, 0, 1, false).
		AddItem(a.logView, 0, 1, false)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftFlex, 0, 1, true).
		AddItem(rightFlex, 0, 2, false)

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
		running := a.isOptionRunning(opt)
		label := formatOptionLabel(opt, running)
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
		var label string
		if opt.ActiveAddons[addon.Name] {
			label = fmt.Sprintf("[green]\u2713 %s %s[-]", addon.Label, addon.Name)
		} else {
			label = fmt.Sprintf("[white]\u2717 %s %s[-]", addon.Label, addon.Name)
		}
		a.addonsList.AddItem(label, "", 0, nil)
	}

	if currentIdx >= len(opt.Addons) {
		currentIdx = len(opt.Addons) - 1
	}
	if currentIdx >= 0 {
		a.addonsList.SetCurrentItem(currentIdx)
	}
}

func formatOptionLabel(opt *Option, running bool) string {
	var b strings.Builder

	// Circle: indicates Docker host status (running/exists)
	if running {
		b.WriteString("[green]\u25cf[-] ")
	} else {
		b.WriteString("[white]\u25cb[-] ")
	}

	// Name color: indicates compose configuration inclusion
	if opt.Enabled {
		b.WriteString(fmt.Sprintf("[green]%s[-]", opt.Name))
	} else {
		b.WriteString(fmt.Sprintf("[white]%s[-]", opt.Name))
	}

	// Addon labels: indicate addon activation status
	for _, addon := range opt.Addons {
		if opt.ActiveAddons[addon.Name] {
			b.WriteString(fmt.Sprintf(" [green](%s)[-]", addon.Label))
		} else {
			b.WriteString(fmt.Sprintf(" [white](%s)[-]", addon.Label))
		}
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
			parts = append(parts, fmt.Sprintf("[white] %s [-]", name))
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
		a.previewView.SetText("[white]No option selected[-]")
		return
	}

	// Always show resolved compose for the selected option
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
	a.statusBar.SetText(" [yellow]j/k[-] nav  [yellow]space[-] toggle  [yellow]e[-] edit  [yellow]U[-]p [yellow]D[-]own=all  [yellow]s[-]top [yellow]c[-]ontinue [yellow]r[-]estart [yellow]p[-]ull  [yellow]SHIFT[-]=all  [yellow]y[-] copy  [yellow]?[-] help  [yellow]q[-] quit")
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

func (a *App) showDockerConfirm(title, message string, borderColor tcell.Color, action func()) {
	a.confirmOpen = true
	a.confirmAction = action

	text := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(message + "\n\n[green]Enter[-] to confirm    [yellow]Esc/q[-] to cancel")

	text.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s ", title)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(borderColor)

	a.pages.AddPage("confirm", modal(text, 55, 9), true, true)
	a.app.SetFocus(text)
}

func (a *App) confirmSingleAction(title, desc string, color tcell.Color, action func()) {
	opt := a.getSelectedOption()
	if opt == nil {
		return
	}
	msg := fmt.Sprintf("[yellow::b]%s[-:-:-]\n\nRun [green]docker %s[-] for [green]%s[-]?", title, desc, opt.Name)
	a.showDockerConfirm(title, msg, color, action)
}

func (a *App) confirmGlobalAction(title, desc string, color tcell.Color, args ...string) {
	msg := fmt.Sprintf("[yellow::b]%s[-:-:-]\n\nRun [green]docker compose %s[-] for all enabled services?", title, desc)
	a.showDockerConfirm(title, msg, color, func() {
		a.dockerComposeGlobal(args...)
	})
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
			"  2             Jump to overrides\n" +
			"  Tab           Cycle panels\n" +
			"  Esc           Back / quit\n\n" +
			"[green]Docker (panel 1, lowercase=service, SHIFT=all):[-]\n" +
			"  U             Up all (start containers)\n" +
			"  D             Down all (remove containers)\n" +
			"  s / S         Stop containers\n" +
			"  c / C         Continue (start stopped)\n" +
			"  r / R         Restart containers\n" +
			"  p / P         Pull images\n\n" +
			"[green]Actions:[-]\n" +
			"  Space / Enter Toggle item\n" +
			"  e             Edit resource file\n" +
			"  y             Copy preview YAML\n" +
			"  Y             Copy global compose\n\n" +
			"[green]Meta:[-]\n" +
			"  q             Quit\n" +
			"  ?             This help\n\n" +
			"[white]Press Escape or q to close[-]")

	helpText.SetBorder(true).
		SetTitle(" Help ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorGreen)

	a.pages.AddPage("help", modal(helpText, 45, 25), true, true)
	a.app.SetFocus(helpText)
}

func (a *App) closeHelp() {
	a.helpOpen = false
	a.pages.RemovePage("help")
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

// --- Clipboard ---

func (a *App) editResourceFile() {
	var filePath string
	if a.currentPanelIdx == 0 {
		opt := a.getSelectedOption()
		if opt == nil {
			return
		}
		filePath = opt.BaseFile
	} else if a.currentPanelIdx == 1 {
		addon := a.getSelectedAddon()
		if addon == nil {
			return
		}
		filePath = addon.File
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	a.app.Suspend(func() {
		cmd := exec.Command(editor, filePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	})
	a.updatePreview()
}

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
