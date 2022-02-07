package main

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os"

	"github.com/octu0/go-xvc"
)

func main() {
	f1, err := os.Open("./testdata/nal_0_16.xvc")
	if err != nil {
		panic(err)
	}
	f2, err := os.Open("./testdata/nal_1_1.xvc")
	if err != nil {
		panic(err)
	}

	data1, err := io.ReadAll(f1)
	if err != nil {
		panic(err)
	}

	data2, err := io.ReadAll(f2)
	if err != nil {
		panic(err)
	}

	decoder, err := xvc.CreateDecoder(
		xvc.DecoderParameterMaxFramerate(30.0),
	)
	if err != nil {
		panic(err)
	}

	if err := decoder.Decode(data1); err != nil {
		panic(err)
	}
	if err := decoder.Decode(data2); err != nil {
		panic(err)
	}

	if decoder.Flush() != true {
		panic("failed to flush")
	}

	i := 0
	for {
		pic, err := decoder.DecodedPicture()
		if err != nil {
			break
		}
		defer pic.Close()

		fmt.Printf("nals[%d] type=%s color_matrix=%s img=%T\n", i, pic.Type(), pic.ColorMatrix(), pic.Image())

		path, err := saveImage(pic.Image())
		if err != nil {
			panic(err)
		}
		fmt.Println("saved", path)
		i += 1
	}
}

func saveImage(img image.Image) (string, error) {
	out, err := ioutil.TempFile("/tmp", "out*.png")
	if err != nil {
		return "", err
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return "", err
	}
	return out.Name(), nil
}
