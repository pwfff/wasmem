package main

import (
	"bytes"
	"image"
	"image/color"
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

type YSorter []color.RGBA

func (r YSorter) Len() int      { return len(r) }
func (r YSorter) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r YSorter) Less(i, j int) bool {
	rgba1 := r[i]
	rgba2 := r[j]

	y1, _, _ := color.RGBToYCbCr(rgba1.R, rgba1.G, rgba1.B)
	y2, _, _ := color.RGBToYCbCr(rgba2.R, rgba2.G, rgba2.B)

	return y1 < y2
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

	var pixels [][]color.RGBA
	for y := 0; y < b.Max.Y; y++ {
		var row []color.RGBA
		for x := 0; x < b.Max.X*4; x += 4 {
			index := y * b.Max.X * 4
			row = append(row, color.RGBA{rgba.Pix[index+x], rgba.Pix[index+x+1], rgba.Pix[index+x+2], rgba.Pix[index+x+3]})
		}
		pixels = append(pixels, row)
	}

	maskRows := getMaskRows(b)

	doSort(rgba, pixels, maskRows)

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

func doSort(baseImage *image.RGBA, basePixels [][]color.RGBA, maskRows [][]sortRow) {
	var wg sync.WaitGroup
	for y := range maskRows {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			for i := range maskRows[y] {
				row := maskRows[y][i]

				sort.Sort(YSorter(basePixels[y][row.start:row.end]))

				for x := row.start; x < row.end; x++ {
					baseImage.Set(x, y, basePixels[y][x])
				}
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
