package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

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
	var preview *widget.Label
	preview = widget.NewLabel("No image selected")
	choose := widget.NewButton("Choose image…", func() {
		dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil { status.SetText(err.Error()); return }
			if r == nil { return }
			selected = r.URI().Path()
			_ = r.Close()
			preview.SetText(filepath.Base(selected))
			status.SetText("Image selected. Choose an output action.")
		}, w).Show()
	})
	buildFLI := widget.NewButton("Export LOGO.FLI", func() {
		if selected == "" { status.SetText("Choose an image first."); return }
		img, err := loadImage(selected); if err != nil { status.SetText(err.Error()); return }
		mode := "pad"; if fit.Selected == "Stretch to 800 × 480" { mode = "stretch" }
		data, err := fli.Encode(img,mode,true); if err != nil { status.SetText(err.Error()); return }
		dialog.NewFileSave(func(out fyne.URIWriteCloser, err error) {
			if err != nil { status.SetText(err.Error()); return }; if out == nil { return }
			_, err = out.Write(data); closeErr := out.Close(); if err == nil { err = closeErr }
			if err != nil { status.SetText(err.Error()) } else { status.SetText("LOGO.FLI exported and checksummed.") }
		},w).Show()
	})
	buildISO := widget.NewButton("Build bootable ISO…", func() {
		if selected == "" { status.SetText("Choose an image first."); return }
		img, err := loadImage(selected); if err != nil { status.SetText(err.Error()); return }
		mode := "pad"; if fit.SelectedIndex() == 1 { mode = "stretch" }
		template := locateTemplate()
		if template == "" { status.SetText("SWL template folder not found. Place template_swl beside the application."); return }
		dialog.NewFileSave(func(out fyne.URIWriteCloser, err error) {
			if err != nil { status.SetText(err.Error()); return }; if out == nil { return }
			path := out.URI().Path(); _ = out.Close()
			if err := iso.Build(img,path,template,mode); err != nil { status.SetText(err.Error()); return }
			status.SetText("ISO created: "+path)
		},w).Show()
	})
	content := container.NewVBox(
		widget.NewLabelWithStyle("VW RNS 510 / Columbus / SEAT Media", fyne.TextAlignLeading, fyne.TextStyle{Bold:true}),
		widget.NewLabel("Go edition • Linux x86-64"),
		container.NewHBox(choose, preview),
		widget.NewForm(widget.NewFormItem("Image scaling",fit)),
		container.NewHBox(buildFLI,buildISO),
		widget.NewSeparator(),
		status,
	)
	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}

func loadImage(path string) (image.Image,error) {
	f,err:=os.Open(path); if err!=nil{return nil,err}; defer f.Close()
	img,_,err:=image.Decode(f); return img,err
}

func locateTemplate() string {
	exe,err:=os.Executable(); if err==nil {
		candidate:=filepath.Join(filepath.Dir(exe),"../share/rns510/template_swl")
		if st,e:=os.Stat(candidate); e==nil && st.IsDir(){return candidate}
	}
	if st,e:=os.Stat("template_swl"); e==nil && st.IsDir(){return "template_swl"}
	return ""
}
