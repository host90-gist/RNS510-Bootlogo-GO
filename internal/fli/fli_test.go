package fli

import (
	"image"
	"image/color"
	"testing"
)

func TestCRC16XMODEMCheckVector(t *testing.T) {
	if got := CRC16XMODEM([]byte("123456789")); got != 0x31c3 {
		t.Fatalf("CRC mismatch: got %04x", got)
	}
}

func TestEncodeAndVerify(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
		}
	}
	data, err := Encode(img, "pad", true)
	if err != nil { t.Fatal(err) }
	report, err := Verify(data)
	if err != nil { t.Fatal(err) }
	if !report.Valid { t.Fatalf("generated FLI did not verify: %+v", report) }
}
