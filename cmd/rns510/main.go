package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/host90-gist/RNS510-Bootlogo-GO/internal/fli"
	"github.com/host90-gist/RNS510-Bootlogo-GO/internal/iso"
)

func usage() {
	fmt.Fprintf(os.Stderr, "RNS510 Toolkit (Go)\n\nUsage:\n  rns510 encode <image> -o <logo.fli>\n  rns510 verify <logo.fli>\n  rns510 build-iso <image> -o <bootlogo.iso> --template <template_swl>\n")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "encode":
		fs := flag.NewFlagSet("encode", flag.ExitOnError)
		out := fs.String("o", "LOGO.FLI", "output FLI path")
		fit := fs.String("fit", "pad", "fit mode: pad or stretch")
		_ = fs.Parse(os.Args[2:])
		if fs.NArg() != 1 {
			usage()
			os.Exit(2)
		}
		if err := encodeImage(fs.Arg(0), *out, *fit); err != nil {
			fatal(err)
		}
	case "verify":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		data, err := os.ReadFile(os.Args[2])
		if err != nil {
			fatal(err)
		}
		report, err := fli.Verify(data)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("FLI: %dx%d, %d pixels\nCRC1: %04x (%v)\nCRC2: %04x (%v)\nRGB555 bit 15 errors: %d\nGreen-noise pixels: %d\n",
			report.Width, report.Height, report.Pixels, report.CRC1, report.CRC1Valid,
			report.CRC2, report.CRC2Valid, report.Bit15Errors, report.GreenNoisePixels)
		if !report.Valid {
			os.Exit(1)
		}
	case "build-iso":
		fs := flag.NewFlagSet("build-iso", flag.ExitOnError)
		out := fs.String("o", "bootlogo.iso", "output ISO path")
		template := fs.String("template", "template_swl", "SWL template directory")
		fit := fs.String("fit", "pad", "fit mode: pad or stretch")
		_ = fs.Parse(os.Args[2:])
		if fs.NArg() != 1 {
			usage()
			os.Exit(2)
		}
		img, err := openImage(fs.Arg(0))
		if err != nil {
			fatal(err)
		}
		if err := iso.Build(img, *out, *template, *fit); err != nil {
			fatal(err)
		}
		fmt.Printf("Created %s\n", filepath.Clean(*out))
	default:
		usage()
		os.Exit(2)
	}
}

func openImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func encodeImage(input, output, fit string) error {
	img, err := openImage(input)
	if err != nil {
		return err
	}
	data, err := fli.Encode(img, fit, true)
	if err != nil {
		return err
	}
	return os.WriteFile(output, data, 0644)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
