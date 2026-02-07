package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/rivo/tview"
)

func highlightCode(code, language string) string {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("gruvbox")
	if style == nil {
		style = styles.Fallback
	}

	var buf strings.Builder
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return tview.Escape(code)
	}

	for token := iterator(); token != chroma.EOF; token = iterator() {
		entry := style.Get(token.Type)
		text := tview.Escape(token.Value)
		if entry.Colour.IsSet() {
			r, g, b := entry.Colour.Red(), entry.Colour.Green(), entry.Colour.Blue()
			buf.WriteString(fmt.Sprintf("[#%02x%02x%02x]%s[-]", r, g, b, text))
		} else {
			buf.WriteString(text)
		}
	}
	return buf.String()
}

func copyToClipboard(text string) error {
	clipboardCmds := []struct {
		name string
		args []string
	}{
		{"wl-copy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
	}

	for _, clip := range clipboardCmds {
		path, err := exec.LookPath(clip.name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path, clip.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no clipboard command available")
}
