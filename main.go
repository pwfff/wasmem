package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"reflect"
	"syscall/js"
	"unsafe"
)

var buf []uint8

func initMem(i []js.Value) {
	length := i[0].Int()
	buf = make([]uint8, length)
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	ptr := uintptr(unsafe.Pointer(hdr.Data))
	log.Println(ptr)
	//js.Global().Set("pointer", ptr)
	js.Global().Call("gotMem", ptr)
}

func processImage([]js.Value) {
	log.Println(buf[:3])
	log.Println(buf[len(buf)-3:])
	r := bytes.NewReader(buf)
	m, _, err := image.Decode(r)

	if err != nil {
		log.Fatal(err)
	}
	bounds := m.Bounds()

	// Calculate a 16-bin histogram for m's red, green, blue and alpha components.
	//
	// An image's bounds do not necessarily start at (0, 0), so the two loops start
	// at bounds.Min.Y and bounds.Min.X. Looping over Y first and X second is more
	// likely to result in better memory access patterns than X first and Y second.
	var histogram [16][4]int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := m.At(x, y).RGBA()
			// A color's RGBA method returns values in the range [0, 65535].
			// Shifting by 12 reduces this to the range [0, 15].
			histogram[r>>12][0]++
			histogram[g>>12][1]++
			histogram[b>>12][2]++
			histogram[a>>12][3]++
		}
	}

	// Print the results.
	fmt.Printf("%-14s %6s %6s %6s %6s\n", "bin", "red", "green", "blue", "alpha")
	for i, x := range histogram {
		fmt.Printf("0x%04x-0x%04x: %6d %6d %6d %6d\n", i<<12, (i+1)<<12-1, x[0], x[1], x[2], x[3])
	}
}

func registerCallbacks() {
	js.Global().Set("initMem", js.NewCallback(initMem))
	js.Global().Set("processImage", js.NewCallback(processImage))
}

func main() {
	c := make(chan struct{}, 0)

	log.Println("WASM Initialized")
	registerCallbacks()
	<-c
}
