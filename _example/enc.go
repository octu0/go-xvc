package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/octu0/go-xvc"
)

func main() {
	f, err := os.Open("./testdata/src.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}

	rgba, err := pngToRGBA(data)
	if err != nil {
		panic(err)
	}

	img := image.NewYCbCr(rgba.Bounds(), image.YCbCrSubsampleRatio420)
	if err := rgbaToYCbCrImage(img, rgba); err != nil {
		panic(err)
	}

	width, height := img.Rect.Dx(), img.Rect.Dy()

	encoder, err := xvc.CreateEncoder(
		xvc.EncoderParameterWidth(width),
		xvc.EncoderParameterHeight(height),
		xvc.EncoderParameterFramerate(30.0),
	)
	if err != nil {
		panic(err)
	}
	defer xvc.DestroyEncoder(encoder)

	userData := time.Now().UnixNano()
	nals, err := encoder.Encode(
		img.Y,       // y plane
		img.Cb,      // u plane
		img.Cr,      // v plane
		img.YStride, // y stride
		img.CStride, // u stride
		img.CStride, // v stride
		userData,    // int64 user_data
	)
	if err != nil {
		panic(err)
	}

	if remainingNals, ok := encoder.Flush(); ok {
		nals = append(nals, remainingNals...)
	}

	for i, nal := range nals {
		defer nal.Close()

		fmt.Printf("nals[%d] type=%s size=%d\n", i, nal.Type(), len(nal.Bytes()))
		out, err := ioutil.TempFile("/tmp", fmt.Sprintf("nal_%d_%d_*.xvc", i, nal.Type()))
		if err != nil {
			panic(err)
		}
		defer out.Close()

		if _, err := out.Write(nal.Bytes()); err != nil {
			panic(err)
		}
		fmt.Println("saved", out.Name())
	}
}

func pngToRGBA(data []byte) (*image.RGBA, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if i, ok := img.(*image.RGBA); ok {
		return i, nil
	}

	b := img.Bounds()
	rgba := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y += 1 {
		for x := b.Min.X; x < b.Max.X; x += 1 {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			rgba.Set(x, y, c)
		}
	}
	return rgba, nil
}

func rgbaToYCbCrImage(dst *image.YCbCr, src *image.RGBA) error {
	rect := src.Bounds()
	width, height := rect.Dx(), rect.Dy()

	for w := 0; w < width; w += 1 {
		for h := 0; h < height; h += 1 {
			c := src.RGBAAt(w, h)
			y, u, v := color.RGBToYCbCr(c.R, c.G, c.B)
			dst.Y[dst.YOffset(w, h)] = y
			dst.Cb[dst.COffset(w, h)] = u
			dst.Cr[dst.COffset(w, h)] = v
		}
	}
	return nil
}
