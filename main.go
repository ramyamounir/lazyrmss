package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	tview.Borders.Horizontal = '─'
	tview.Borders.Vertical = '│'
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'
}

type Category struct {
	Name string
	Dir  string
}

type Option struct {
	Name         string
	Dir          string
	Category     string
	BaseFile     string
	Addons       []Addon
	Enabled      bool
	ActiveAddons map[string]bool
}

type Addon struct {
	Name  string
	File  string
	Label string
	Color string
}

type addonDisplay struct {
	Label string
	Color string
}

var addonDisplayMap = map[string]addonDisplay{
	"network": {Label: "N", Color: "blue"},
	"gpu":     {Label: "G", Color: "magenta"},
}

func getAddonDisplay(name string) (string, string) {
	if d, ok := addonDisplayMap[name]; ok {
		return d.Label, d.Color
	}
	return strings.ToUpper(name[:1]), "yellow"
}

type App struct {
	app             *tview.Application
	pages           *tview.Pages
	panels          []tview.Primitive
	currentPanelIdx int

	tabBar      *tview.TextView
	optionsList *tview.List
	addonsList  *tview.List
	previewView *tview.TextView
	logView     *tview.TextView
	statusBar   *tview.TextView

	helpOpen      bool
	confirmOpen   bool
	confirmAction func()

	config       *Config
	categories   []Category
	activeTabIdx int
	options      map[string][]*Option

	dockerStatus *DockerStatus
	dockerCancel context.CancelFunc
}

func main() {
	a := &App{
		options: make(map[string][]*Option),
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	a.config = cfg

	if err := a.discoverAll(); err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering services: %v\n", err)
		os.Exit(1)
	}

	if err := a.loadState(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load state: %v\n", err)
	}

	a.setupUI()
	a.refreshAll()

	// Initialize Docker status polling
	a.dockerStatus = &DockerStatus{
		RunningContainers: make(map[string]bool),
		ExistingNetworks:  make(map[string]bool),
		ExistingVolumes:   make(map[string]bool),
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.dockerCancel = cancel

	interval := time.Duration(a.config.PollInterval) * time.Second
	a.dockerStatus.StartPolling(ctx, interval, func() {
		a.app.QueueUpdateDraw(func() {
			a.refreshOptionsList()
		})
	})

	if err := a.app.Run(); err != nil {
		cancel()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	cancel()
}

// --- Panel navigation ---

func (a *App) focusPanel(idx int) {
	if idx >= 0 && idx < len(a.panels) {
		a.currentPanelIdx = idx
		a.app.SetFocus(a.panels[idx])
		a.updateBorderColors()
	}
}

func (a *App) nextPanel() {
	a.currentPanelIdx = (a.currentPanelIdx + 1) % len(a.panels)
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

func (a *App) prevPanel() {
	a.currentPanelIdx = (a.currentPanelIdx - 1 + len(a.panels)) % len(a.panels)
	a.app.SetFocus(a.panels[a.currentPanelIdx])
	a.updateBorderColors()
}

func (a *App) updateBorderColors() {
	selectionColor := tcell.NewRGBColor(68, 68, 88)

	for _, p := range a.panels {
		if box, ok := p.(interface {
			SetBorderColor(tcell.Color) *tview.Box
		}); ok {
			box.SetBorderColor(tcell.ColorDefault)
		}
		if list, ok := p.(*tview.List); ok {
			list.SetSelectedStyle(tcell.StyleDefault)
		}
	}

	if a.currentPanelIdx < len(a.panels) {
		focused := a.panels[a.currentPanelIdx]
		if box, ok := focused.(interface {
			SetBorderColor(tcell.Color) *tview.Box
		}); ok {
			box.SetBorderColor(tcell.ColorGreen)
		}
		if list, ok := focused.(*tview.List); ok {
			list.SetSelectedStyle(tcell.StyleDefault.Background(selectionColor))
		}
	}
}

// --- Cursor movement ---

func (a *App) cursorDown() {
	if list, ok := a.panels[a.currentPanelIdx].(*tview.List); ok {
		count := list.GetItemCount()
		current := list.GetCurrentItem()
		if current < count-1 {
			list.SetCurrentItem(current + 1)
		}
	}
	if a.currentPanelIdx == 0 {
		a.refreshAddonsList()
		a.updatePanelTitles()
	}
	a.updatePreview()
}

func (a *App) cursorUp() {
	if list, ok := a.panels[a.currentPanelIdx].(*tview.List); ok {
		current := list.GetCurrentItem()
		if current > 0 {
			list.SetCurrentItem(current - 1)
		}
	}
	if a.currentPanelIdx == 0 {
		a.refreshAddonsList()
		a.updatePanelTitles()
	}
	a.updatePreview()
}

func (a *App) scrollPreviewDown() {
	row, col := a.previewView.GetScrollOffset()
	a.previewView.ScrollTo(row+1, col)
}

func (a *App) scrollPreviewUp() {
	row, col := a.previewView.GetScrollOffset()
	if row > 0 {
		a.previewView.ScrollTo(row-1, col)
	}
}

// --- Tab navigation ---

func (a *App) nextTab() {
	if len(a.categories) == 0 {
		return
	}
	a.activeTabIdx = (a.activeTabIdx + 1) % len(a.categories)
	a.refreshAll()
}

func (a *App) prevTab() {
	if len(a.categories) == 0 {
		return
	}
	a.activeTabIdx = (a.activeTabIdx - 1 + len(a.categories)) % len(a.categories)
	a.refreshAll()
}
