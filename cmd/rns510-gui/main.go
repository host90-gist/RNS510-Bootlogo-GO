package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/host90-gist/RNS510-Bootlogo-GO/internal/fli"
	"github.com/host90-gist/RNS510-Bootlogo-GO/internal/iso"
)

func main() {
	a := app.NewWithID("org.rns510.toolkit")
	w := a.NewWindow("RNS 510 Bootlogo Toolkit")
	w.Resize(fyne.NewSize(720, 420))

	status := widget.NewLabel("Choose an image to create a LOGO.FLI or bootable SWL ISO.")
	fit := widget.NewSelect([]string{"Preserve aspect ratio (black bars)", "Stretch to 800 × 480"}, nil)
	fit.SetSelected("Preserve aspect ratio (black bars)")

	var selected string
	preview := widget.NewLabel("No image selected")
	choose := widget.NewButton("Chose Image for Bootlogo", func() {
		start := userHome()
		if selected != "" {
			start = filepath.Dir(selected)
		}
		showFilePicker(w, "Chose Image for Bootlogo", start, false, "", []string{".png", ".jpg", ".jpeg"}, func(path string) {
			selected = path
			preview.SetText(filepath.Base(selected))
			status.SetText("Image selected. Choose an output action.")
		})
	})

	buildFLI := widget.NewButton("Export LOGO.FLI", func() {
		if selected == "" {
			dialog.ShowInformation("No image selected", "Choose an image first.", w)
			return
		}
		img, err := loadImage(selected)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		mode := selectedFit(fit)
		showFilePicker(w, "Save LOGO.FLI", userHome(), true, "LOGO.FLI", []string{".fli"}, func(path string) {
			data, err := fli.Encode(img, mode, true)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if err := os.WriteFile(path, data, 0644); err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("LOGO.FLI is generated", "File location:\n"+path, w)
			status.SetText("LOGO.FLI generated: " + path)
		})
	})

	buildISO := widget.NewButton("Build bootable ISO…", func() {
		if selected == "" {
			dialog.ShowInformation("No image selected", "Choose an image first.", w)
			return
		}
		img, err := loadImage(selected)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		template := locateTemplate()
		if template == "" {
			dialog.ShowError(fmt.Errorf("SWL template folder not found. Place template_swl beside the application."), w)
			return
		}
		showFilePicker(w, "Save bootable ISO", userHome(), true, "bootlogo.iso", []string{".iso"}, func(path string) {
			if err := iso.Build(img, path, template, selectedFit(fit)); err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("ISO is generated", "File location:\n"+path, w)
			status.SetText("ISO generated: " + path)
		})
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("VW RNS 510 / Columbus / SEAT Media", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Go edition • Linux x86-64"),
		container.NewHBox(choose, preview),
		widget.NewForm(widget.NewFormItem("Image scaling", fit)),
		container.NewHBox(buildFLI, buildISO),
		status,
	)
	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}

func selectedFit(fit *widget.Select) string {
	if fit.Selected == "Stretch to 800 × 480" {
		return "stretch"
	}
	return "pad"
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func userHome() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}
	return "."
}

// showFilePicker provides a resizable file browser with live filename filtering.
// Directories remain navigable; typing filters the current directory by basename.
func showFilePicker(parent fyne.Window, title, startDir string, save bool, defaultName string, extensions []string, onChoose func(string)) {
	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		currentDir = userHome()
	}
	if info, statErr := os.Stat(currentDir); statErr != nil || !info.IsDir() {
		currentDir = userHome()
	}

	browser := fyne.CurrentApp().NewWindow(title)
	browser.Resize(fyne.NewSize(760, 540))

	location := widget.NewLabel(currentDir)
	search := widget.NewEntry()
	search.SetPlaceHolder("Search files by name…")
	filename := widget.NewEntry()
	filename.SetPlaceHolder("File name")
	if save {
		filename.SetText(defaultName)
	}

	var entries []os.DirEntry
	var selectedPath string
	var visible []os.DirEntry
	var list *widget.List

	allowed := func(name string) bool {
		if len(extensions) == 0 {
			return true
		}
		ext := strings.ToLower(filepath.Ext(name))
		for _, candidate := range extensions {
			if ext == candidate {
				return true
			}
		}
		return false
	}

	refresh := func() {}
	refresh = func() {
		read, readErr := os.ReadDir(currentDir)
		if readErr != nil {
			dialog.ShowError(readErr, browser)
			return
		}
		entries = read
		visible = visible[:0]
		query := strings.ToLower(strings.TrimSpace(search.Text))
		for _, item := range entries {
			if item.IsDir() {
				if query == "" || strings.Contains(strings.ToLower(item.Name()), query) {
					visible = append(visible, item)
				}
				continue
			}
			if !allowed(item.Name()) {
				continue
			}
			if query == "" || strings.Contains(strings.ToLower(item.Name()), query) {
				visible = append(visible, item)
			}
		}
		location.SetText(currentDir)
		if list != nil {
			list.Refresh()
		}
	}

	list = widget.NewList(
		func() int { return len(visible) },
		func() fyne.CanvasObject { return widget.NewLabel("filename") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(visible) {
				return
			}
			item := visible[id]
			label := item.Name()
			if item.IsDir() {
				label = "📁 " + label + "/"
			}
			obj.(*widget.Label).SetText(label)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(visible) {
			return
		}
		item := visible[id]
		path := filepath.Join(currentDir, item.Name())
		if item.IsDir() {
			currentDir = path
			selectedPath = ""
			list.UnselectAll()
			refresh()
			return
		}
		selectedPath = path
		if save {
			filename.SetText(item.Name())
		}
	}
	search.OnChanged = func(string) {
		selectedPath = ""
		refresh()
	}

	up := widget.NewButton("Up one folder", func() {
		parentDir := filepath.Dir(currentDir)
		if parentDir != currentDir {
			currentDir = parentDir
			selectedPath = ""
			refresh()
		}
	})

	chooseLabel := "Open"
	if save {
		chooseLabel = "Save"
	}
	choose := widget.NewButton(chooseLabel, func() {
		path := selectedPath
		if save {
			name := strings.TrimSpace(filename.Text)
			if name == "" {
				dialog.ShowInformation("Filename required", "Enter a filename to save.", browser)
				return
			}
			if filepath.Ext(name) == "" && len(extensions) > 0 {
				name += extensions[0]
			}
			if !allowed(name) {
				dialog.ShowInformation("Unsupported file type", "Choose a filename with a supported extension.", browser)
				return
			}
			if filepath.IsAbs(name) {
				path = filepath.Clean(name)
			} else {
				path = filepath.Join(currentDir, name)
			}
		}
		if path == "" {
			dialog.ShowInformation("No file selected", "Select a file first.", browser)
			return
		}
		if !save {
			info, statErr := os.Stat(path)
			if statErr != nil || info.IsDir() {
				dialog.ShowInformation("Invalid selection", "Select a valid file.", browser)
				return
			}
		}
		if save {
			if _, statErr := os.Stat(path); statErr == nil {
				dialog.ShowConfirm("Overwrite file?", "The selected file already exists. Overwrite it?", func(ok bool) {
					if ok {
						browser.Close()
						onChoose(path)
					}
				}, browser)
				return
			}
		}
		browser.Close()
		onChoose(path)
	})

	cancel := widget.NewButton("Cancel", func() { browser.Close() })
	top := container.NewVBox(location, container.NewBorder(nil, nil, nil, up, search))
	bottomItems := []fyne.CanvasObject{}
	if save {
		bottomItems = append(bottomItems, widget.NewLabel("Save as:"), filename)
	}
	bottomItems = append(bottomItems, choose, cancel)
	browser.SetContent(container.NewBorder(top, container.NewHBox(bottomItems...), nil, nil, list))
	browser.Show()
	refresh()
}

func locateTemplate() string {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "../share/rns510/template_swl")
		if st, e := os.Stat(candidate); e == nil && st.IsDir() {
			return candidate
		}
	}
	if st, e := os.Stat("template_swl"); e == nil && st.IsDir() {
		return "template_swl"
	}
	return ""
}
