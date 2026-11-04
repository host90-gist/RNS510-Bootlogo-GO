// Package fli implements the RNS 510 LOGO.FLI RGB555/RLE container.
package fli

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
)

const (
	Width  = 800
	Height = 480
	HeaderSize = 64
	Magic uint32 = 0xAA55AA55
)

var ErrInvalidFLI = errors.New("invalid RNS 510 FLI file")

func CRC16XMODEM(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// Encode converts an image to the 64-byte OEM-style header and big-endian
// RGB555 RLE stream. fit is either "pad" (letterbox) or "stretch".
func Encode(src image.Image, fit string, cleanBlack bool) ([]byte, error) {
	if src == nil {
		return nil, errors.New("no input image")
	}
	if fit != "pad" && fit != "stretch" {
		return nil, fmt.Errorf("unknown fit mode %q (use pad or stretch)", fit)
	}
	pixels := make([]uint16, Width*Height)
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw <= 0 || sh <= 0 {
		return nil, errors.New("input image has empty dimensions")
	}
	dw, dh, ox, oy := Width, Height, 0, 0
	if fit == "pad" {
		scale := math.Min(float64(Width)/float64(sw), float64(Height)/float64(sh))
		dw = max(1, int(math.Round(float64(sw)*scale)))
		dh = max(1, int(math.Round(float64(sh)*scale)))
		ox, oy = (Width-dw)/2, (Height-dh)/2
	}
	for y := 0; y < dh; y++ {
		sy := src.Bounds().Min.Y + min(sh-1, y*sh/dh)
		for x := 0; x < dw; x++ {
			sx := src.Bounds().Min.X + min(sw-1, x*sw/dw)
			c := color.NRGBAModel.Convert(src.At(sx, sy)).(color.NRGBA)
			// Composite transparency over black, matching the original tool.
			r := uint8(uint16(c.R)*uint16(c.A)/255)
			g := uint8(uint16(c.G)*uint16(c.A)/255)
			b := uint8(uint16(c.B)*uint16(c.A)/255)
			var word uint16
			if !(cleanBlack && r < 10 && g < 10 && b < 10) {
				r5, g5, b5 := r>>3, g>>3, b>>3
				if cleanBlack && r5 == 0 && b5 == 0 && g5 <= 1 {
					g5 = 0
				}
				word = uint16(r5)<<10 | uint16(g5)<<5 | uint16(b5)
			}
			pixels[(oy+y)*Width+ox+x] = word
		}
	}
	rle := make([]byte, 0, len(pixels)/2)
	current, count := pixels[0], uint16(1)
	flush := func() {
		var pair [4]byte
		binary.BigEndian.PutUint16(pair[0:2], current)
		binary.BigEndian.PutUint16(pair[2:4], count)
		rle = append(rle, pair[:]...)
	}
	for _, p := range pixels[1:] {
		if p == current && count < 0xffff {
			count++
			continue
		}
		flush()
		current, count = p, 1
	}
	flush()
	return BuildContainer(rle, Width, Height), nil
}

// BuildContainer writes the header layout used by the original serializer:
// CRC1 at bytes 18..19, CRC2 at 50..51, dimensions at 40..47,
// and RLE byte length at 60..63.
func BuildContainer(rle []byte, width, height int) []byte {
	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(header[0:4], Magic)
	binary.BigEndian.PutUint32(header[4:8], 15)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(rle)+32))
	binary.BigEndian.PutUint32(header[32:36], 0x002A4444)
	binary.BigEndian.PutUint32(header[36:40], 0x00021675)
	binary.BigEndian.PutUint32(header[40:44], uint32(width))
	binary.BigEndian.PutUint32(header[44:48], uint32(height))
	binary.BigEndian.PutUint16(header[48:50], 0)
	binary.BigEndian.PutUint16(header[50:52], CRC16XMODEM(rle))
	binary.BigEndian.PutUint32(header[52:56], 1)
	binary.BigEndian.PutUint32(header[56:60], 0)
	binary.BigEndian.PutUint32(header[60:64], uint32(len(rle)))
	crcInput := append(append([]byte{}, header[32:64]...), rle...)
	binary.BigEndian.PutUint16(header[18:20], CRC16XMODEM(crcInput))
	return append(header, rle...)
}

type Report struct {
	Width, Height int
	Pixels int
	CRC1, CRC2 uint16
	CRC1Valid, CRC2Valid bool
	Bit15Errors, GreenNoisePixels int
	Valid bool
}

func Verify(data []byte) (Report, error) {
	var r Report
	if len(data) < HeaderSize || (len(data)-HeaderSize)%4 != 0 {
		return r, ErrInvalidFLI
	}
	h, rle := data[:HeaderSize], data[HeaderSize:]
	if binary.BigEndian.Uint32(h[:4]) != Magic {
		return r, fmt.Errorf("%w: bad magic", ErrInvalidFLI)
	}
	r.Width = int(binary.BigEndian.Uint32(h[40:44]))
	r.Height = int(binary.BigEndian.Uint32(h[44:48]))
	if r.Width <= 0 || r.Height <= 0 || r.Width > 8192 || r.Height > 8192 {
		return r, fmt.Errorf("%w: unreasonable dimensions", ErrInvalidFLI)
	}
	if int(binary.BigEndian.Uint32(h[60:64])) != len(rle) {
		return r, fmt.Errorf("%w: payload length mismatch", ErrInvalidFLI)
	}
	r.CRC1 = binary.BigEndian.Uint16(h[18:20])
	r.CRC2 = binary.BigEndian.Uint16(h[50:52])
	r.CRC2Valid = r.CRC2 == CRC16XMODEM(rle)
	crcInput := append(append([]byte{}, h[32:64]...), rle...)
	r.CRC1Valid = r.CRC1 == CRC16XMODEM(crcInput)
	for i := 0; i < len(rle); i += 4 {
		word := binary.BigEndian.Uint16(rle[i:i+2])
		count := int(binary.BigEndian.Uint16(rle[i+2:i+4]))
		if count == 0 {
			return r, fmt.Errorf("%w: zero-length RLE run", ErrInvalidFLI)
		}
		r.Pixels += count
		if word&0x8000 != 0 {
			r.Bit15Errors += count
		}
		red, green, blue := int((word>>10)&31), int((word>>5)&31), int(word&31)
		if (green > 0 && red == 0 && blue == 0 && green <= 3) ||
			(green > red+3 && green > blue+3 && red+green+blue < 18) {
			r.GreenNoisePixels += count
		}
	}
	r.Valid = r.Width == Width && r.Height == Height && r.Pixels == Width*Height &&
		r.CRC1Valid && r.CRC2Valid && r.Bit15Errors == 0 && r.GreenNoisePixels == 0
	return r, nil
}
