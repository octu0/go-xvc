package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"time"

	"github.com/octu0/go-xvc"
)

func main() {
	var input string
	var width int
	var height int
	flag.StringVar(&input, "i", "", "input yuv file path")
	flag.IntVar(&width, "w", 152, "source width")
	flag.IntVar(&height, "h", 100, "source height")
	flag.Parse()

	f, err := os.Open(input)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	encoder, err := xvc.CreateEncoder(
		xvc.EncoderParameterWidth(width),
		xvc.EncoderParameterHeight(height),
		xvc.EncoderParameterFramerate(30.0),
	)
	if err != nil {
		panic(err)
	}
	defer xvc.DestroyEncoder(encoder)

	decoder, err := xvc.CreateDecoder(
		xvc.DecoderParameterMaxFramerate(30.0),
	)
	if err != nil {
		panic(err)
	}
	defer xvc.DestroyDecoder(decoder)

	// I420: 1(Y) + 1/4(U) + 1/4(V)
	frameSize := (width * height) * 3 / 2
	buf := make([]byte, frameSize)

	frames := 0
	br := bufio.NewReader(f)
	t := time.Now()
	for numBytes, _ := br.Read(buf); numBytes == frameSize; numBytes, _ = br.Read(buf) {
		s0 := width * height
		s1 := s0 >> 2
		y := make([]byte, s0)
		u := make([]byte, s1)
		v := make([]byte, s1)
		copy(y, buf[0:s0])
		copy(u, buf[s0:s0+s1])
		copy(v, buf[s0+s1:s0+s1+s1])

		since := time.Since(t)
		nals, err := encoder.Encode(
			y,            // y plane
			u,            // u plane
			v,            // v plane
			width,        // y stride
			width/2,      // u stride
			width/2,      // v stride
			int64(since), // int64 user_data
		)
		if err != nil {
			panic(err)
		}

		if (frames % 3) == 0 {
			if remainingNals, ok := encoder.Flush(); ok {
				nals = append(nals, remainingNals...)
			}
		}

		for _, nal := range nals {
			if err := decoder.Decode(nal.Bytes()); err != nil {
				panic(err)
			}
		}

		if (frames % 3) == 0 {
			decoder.Flush()
		}

		frames += 1

		pic, err := decoder.DecodedPicture()
		if err != nil {
			continue
		}
		defer pic.Close()

		fmt.Printf("frame[%d] type=%s color_matrix=%d img=%T\n", frames, pic.Type(), pic.ColorMatrix(), pic.Image())

		path, err := saveImage(pic.Image())
		if err != nil {
			panic(err)
		}
		fmt.Println("saved", path)
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
