//go:build windows

package applications

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"unsafe"

	"golang.org/x/sys/windows"
)

type bitmapInfo struct {
	Size                   uint32
	Width, Height          int32
	Planes, BitCount       uint16
	Compression, SizeImage uint32
	XPels, YPels           int32
	Used, Important        uint32
}

func Icon(path string) string {
	shell := windows.NewLazySystemDLL("shell32.dll")
	gdi := windows.NewLazySystemDLL("gdi32.dll")
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var icon uintptr
	n, _, _ := shell.NewProc("ExtractIconExW").Call(uintptr(unsafe.Pointer(p)), 0, 0, uintptr(unsafe.Pointer(&icon)), 1)
	if n == 0 || icon == 0 {
		return ""
	}
	defer user32.NewProc("DestroyIcon").Call(icon)
	dc, _, _ := gdi.NewProc("CreateCompatibleDC").Call(0)
	if dc == 0 {
		return ""
	}
	defer gdi.NewProc("DeleteDC").Call(dc)
	info := bitmapInfo{Size: 40, Width: 32, Height: -32, Planes: 1, BitCount: 32}
	var bits unsafe.Pointer
	bm, _, _ := gdi.NewProc("CreateDIBSection").Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bm == 0 || bits == nil {
		return ""
	}
	defer gdi.NewProc("DeleteObject").Call(bm)
	old, _, _ := gdi.NewProc("SelectObject").Call(dc, bm)
	defer gdi.NewProc("SelectObject").Call(dc, old)
	user32.NewProc("DrawIconEx").Call(dc, 0, 0, icon, 32, 32, 0, 0, 3)
	data := unsafe.Slice((*byte)(bits), 32*32*4)
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := 0; i < len(data); i += 4 {
		img.Pix[i] = data[i+2]
		img.Pix[i+1] = data[i+1]
		img.Pix[i+2] = data[i]
		img.Pix[i+3] = data[i+3]
		if data[i+3] == 0 && (data[i]|data[i+1]|data[i+2]) != 0 {
			img.Pix[i+3] = 255
		}
	}
	var buf bytes.Buffer
	if png.Encode(&buf, img) != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
