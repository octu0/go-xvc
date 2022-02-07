# `go-xvc`

[![MIT License](https://img.shields.io/github/license/octu0/go-xvc)](https://github.com/octu0/go-xvc/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/octu0/go-xvc?status.svg)](https://godoc.org/github.com/octu0/go-xvc)
[![Go Report Card](https://goreportcard.com/badge/github.com/octu0/go-xvc)](https://goreportcard.com/report/github.com/octu0/go-xvc)
[![Releases](https://img.shields.io/github/v/release/octu0/go-xvc)](https://github.com/octu0/go-xvc/releases)

Go bindings for [divideon/xvc](https://github.com/divideon/xvc)  
Available for encode/decode of xvc video codec.

## Requirements

requires xvc [install](https://github.com/divideon/xvc#linux-build-steps) on your system

```
$ git clone https://github.com/divideon/xvc.git
$ cd xvc
$ mkdir build
$ cd build
$ cmake ..
$ make
$ make install
```

## Usage

### Encode

```go
import "github.com/octu0/go-xvc"

func encode(out io.Writer) {
	encoder, err := xvc.CreateEncoder(
		xvc.EncoderParameterWidth(width),
		xvc.EncoderParameterHeight(height),
		xvc.EncoderParameterFramerate(30.0),
	)
	if err != nil {
		panic(err)
	}
	defer xvc.DestroyEncoder(encoder)

	var userData int64
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

	for _, nal := range nals {
		defer nal.Close()

		if _, err := out.Write(nal.Bytes()); err != nil {
			panic(err)
		}
	}
}
```

### Decode

```go
import "github.com/octu0/go-xvc"

func decode(in io.Reader) {
	decoder, err := xvc.CreateDecoder(
		xvc.DecoderParameterMaxFramerate(30.0),
	)
	if err != nil {
		panic(err)
	}
	defer xvc.DestroyEncoder(decoder)

	data, err := io.ReadAll(in)
	if err != nil {
		panic(err)
	}
	if err := decoder.Decode(data); err != nil {
		panic(err)
	}

	if decoder.Flush(); != true {
		panic("failed to flush")
	}

	for {
		pic, err := decoder.DecodedPicture()
		if err != nil {
			break
		}
		defer pic.Close()

		fmt.Printf(
			"type=%s color_matrix=%d img=%T\n", 
			pic.Type(), pic.ColorMatrix(), pic.Image(),
		) // => type=intra_access_picture color_matrix=3 img=*image.YCbCr
	}
}
```
