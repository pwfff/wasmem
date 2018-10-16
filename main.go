package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"log"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"syscall/js"
	"unsafe"
)

type sortRow struct {
	start int
	end   int
}

type YSorter []uint8

func (r YSorter) Len() int { return len(r) / 4 }
func (r YSorter) Swap(i, j int) {
	i = i * 4
	j = j * 4
	for o := 0; o < 4; o++ {
		r[i+o], r[j+o] = r[j+o], r[i+o]
	}
}
func (r YSorter) Less(i, j int) bool {
	i = i * 4
	j = j * 4

	y1 := getY(r[i], r[i+1], r[i+2])
	y2 := getY(r[j], r[j+1], r[j+2])

	return y1 < y2
}

func getY(r, g, b uint8) uint8 {
	return (r + r + r + b + g + g + g + g) >> 3
}

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
	r := bytes.NewReader(buf)
	src, _, err := image.Decode(r)

	if err != nil {
		log.Fatal(err)
	}

	b := src.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, src, b.Min, draw.Src)

	maskRows := getMaskRows(b)

	doSort(rgba, maskRows)

	w := new(bytes.Buffer)

	jpeg.Encode(w, rgba, &jpeg.Options{100})
	out := w.Bytes()

	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&out))
	ptr := uintptr(unsafe.Pointer(hdr.Data))

	js.Global().Call("imageProcessed", ptr, len(out))
}

func getMaskRows(bounds image.Rectangle) [][]sortRow {
	width, height := bounds.Max.X, bounds.Max.Y

	var maskRows [][]sortRow
	for y := 0; y < height; y++ {
		var rows []sortRow
		var start int
		var inRow bool

		doRandom := true

		for x := 0; x < width; x++ {
			if !inRow {
				start = x
				inRow = true
			} else {
				if doRandom && rand.Float64() > .9 {
					rows = append(rows, sortRow{start, x})
					inRow = false
				}
			}
		}
		maskRows = append(maskRows, rows)
	}

	return maskRows
}

func doSort(baseImage *image.RGBA, maskRows [][]sortRow) {
	var wg sync.WaitGroup
	for y := range maskRows {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			for i := range maskRows[y] {
				row := maskRows[y][i]

				// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*4].
				start := (y-baseImage.Rect.Min.Y)*baseImage.Stride + (row.start-baseImage.Rect.Min.X)*4
				end := start + (row.end-row.start)*4
				sort.Sort(YSorter(baseImage.Pix[start:end]))
			}
		}(y)
	}
	wg.Wait()
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
