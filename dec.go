package xvc

/*
#cgo CFLAGS: -I${SRCDIR}/include -I/usr/local/include -I/usr/include
#cgo darwin LDFLAGS: -L${SRCDIR} -L/usr/local/lib -L/usr/lib -lxvcdec -lc++
#cgo linux  LDFLAGS: -L${SRCDIR} -L/usr/local/lib -L/usr/lib -lxvcdec -lc++
#include <stdint.h>
#include <stdlib.h>

#include "dec.h"
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"runtime"
	"sync/atomic"
	"unsafe"
)

type DecodedPicture struct {
	width, height int
	nalType       NALUnitType
	colorMatrix   ColorMatrix
	img           image.Image
	userData      int64
	closed        int32
	closeFunc     func()
}

func (n *DecodedPicture) Close() {
	if atomic.CompareAndSwapInt32(&n.closed, 0, 1) {
		n.closeFunc()
	}
}

func (n *DecodedPicture) Width() int {
	return n.width
}

func (n *DecodedPicture) Height() int {
	return n.height
}

func (n *DecodedPicture) Type() NALUnitType {
	return n.nalType
}

func (n *DecodedPicture) ColorMatrix() ColorMatrix {
	return n.colorMatrix
}

func (n *DecodedPicture) Image() image.Image {
	return n.img
}

func (n *DecodedPicture) UserData() int64 {
	return n.userData
}

type decoderParameterFunc func(*decoderParameter)
type decoderParameter struct {
	width          int
	height         int
	chromaFormat   ChromaFormat
	colorMatrix    ColorMatrix
	maxFramerate   float32
	threads        int // -1: auto-detect
	bitDepth       int
	bufferPoolFunc func() BufferPool
}

func (d *decoderParameter) setCParam(param *C.xvc_decoder_parameters) {
	param.output_width = C.int(d.width)
	param.output_height = C.int(d.height)
	param.max_framerate = C.double(d.maxFramerate)
	param.threads = C.int(d.threads)
	param.output_bitdepth = C.int(d.bitDepth)

	switch d.chromaFormat {
	case ChromaFormatMonochrome:
		param.output_chroma_format = C.XVC_DEC_CHROMA_FORMAT_MONOCHROME
	case ChromaFormat420:
		param.output_chroma_format = C.XVC_DEC_CHROMA_FORMAT_420
	case ChromaFormat422:
		param.output_chroma_format = C.XVC_DEC_CHROMA_FORMAT_422
	case ChromaFormat444:
		param.output_chroma_format = C.XVC_DEC_CHROMA_FORMAT_444
	case ChromaFormatUnified:
		param.output_chroma_format = C.XVC_DEC_CHROMA_FORMAT_UNDEFINED
	}

	switch d.colorMatrix {
	case ColorMatrix601:
		param.output_color_matrix = C.XVC_DEC_COLOR_MATRIX_601
	case ColorMatrix709:
		param.output_color_matrix = C.XVC_DEC_COLOR_MATRIX_709
	case ColorMatrix2020:
		param.output_color_matrix = C.XVC_DEC_COLOR_MATRIX_2020
	case ColorMatrixUnified:
		param.output_color_matrix = C.XVC_DEC_COLOR_MATRIX_UNDEFINED
	}
}

func defaultDecoderParameter() *decoderParameter {
	return &decoderParameter{
		chromaFormat: ChromaFormat420,
		colorMatrix:  ColorMatrix2020,
		threads:      -1, // auto
		bitDepth:     8,  // 8bit
		bufferPoolFunc: func() BufferPool {
			return newSimpleBufferPool(4 * 1024)
		},
	}
}

func DecoderParameterWidth(width int) decoderParameterFunc {
	return func(p *decoderParameter) {
		p.width = width
	}
}

func DecoderParameterHeight(height int) decoderParameterFunc {
	return func(p *decoderParameter) {
		p.height = height
	}
}

func DecoderParameterMaxFramerate(rate float32) decoderParameterFunc {
	return func(p *decoderParameter) {
		p.maxFramerate = rate
	}
}

func DecoderBufferPool(fn func() BufferPool) decoderParameterFunc {
	return func(p *decoderParameter) {
		p.bufferPoolFunc = fn
	}
}

type Decoder struct {
	api     unsafe.Pointer // xvc_decoder_api*
	decoder unsafe.Pointer // xvc_decoder*
	pool    BufferPool
}

func (d *Decoder) Decode(nalData []byte) error {
	r := bytes.NewReader(nalData[0:4])

	nalSize := [4]uint8{}
	if err := binary.Read(r, binary.BigEndian, &nalSize); err != nil {
		return err
	}

	length := uint32(nalSize[0]) | uint32(nalSize[1])<<8 | uint32(nalSize[2])<<16 | uint32(nalSize[3])<<24
	data := nalData[4:length]

	ret := C.decoder_decode_nal(
		(*C.xvc_decoder_api)(d.api),
		(*C.xvc_decoder)(d.decoder),
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.size_t(length),
		C.int64_t(0),
	)

	switch ret {
	case C.XVC_DEC_BITSTREAM_VERSION_LOWER_THAN_SUPPORTED_BY_DECODER,
		C.XVC_DEC_BITSTREAM_VERSION_HIGHER_THAN_DECODER,
		C.XVC_DEC_BITSTREAM_BITDEPTH_TOO_HIGH:
		return DecReturnCode(ret)
	}
	return nil
}

func (d *Decoder) Flush() bool {
	ret := C.decoder_flush(
		(*C.xvc_decoder_api)(d.api),
		(*C.xvc_decoder)(d.decoder),
	)
	if ret != C.XVC_DEC_OK {
		return false
	}
	return true
}

func (d *Decoder) DecodedPicture() (*DecodedPicture, error) {
	pic := C.decoder_picture_create(
		(*C.xvc_decoder_api)(d.api),
		(*C.xvc_decoder)(d.decoder),
	)
	defer C.decoder_picture_destroy(
		(*C.xvc_decoder_api)(d.api),
		pic,
	)

	ret := C.decoder_get_picture(
		(*C.xvc_decoder_api)(d.api),
		(*C.xvc_decoder)(d.decoder),
		pic,
	)
	if ret != C.XVC_DEC_OK {
		return nil, DecReturnCode(ret)
	}

	buf := d.pool.Get()
	buf.Grow(int(pic.size))
	buf.Write(C.GoBytes(unsafe.Pointer(pic.bytes), C.int(pic.size)))

	width, height := int(pic.stats.width), int(pic.stats.height)
	img := d.createImage(buf, width, height, ChromaFormat(pic.stats.chroma_format))

	dpic := &DecodedPicture{
		width:       width,
		height:      height,
		nalType:     NALUnitType(pic.stats.nal_unit_type),
		colorMatrix: ColorMatrix(pic.stats.color_matrix),
		img:         img,
		userData:    int64(pic.user_data),
		closed:      int32(0),
		closeFunc: func() {
			buf.Reset()
			d.pool.Put(buf)
		},
	}
	return dpic, nil
}

func (d *Decoder) createImage(buf *bytes.Buffer, width, height int, format ChromaFormat) image.Image {
	switch format {
	case ChromaFormat420:
		return d.yuvImage(buf, width, height, image.YCbCrSubsampleRatio420)
	case ChromaFormat444:
		return d.yuvImage(buf, width, height, image.YCbCrSubsampleRatio444)
	default:
		panic(fmt.Sprintf("unsupport format: %d", format))
	}
}

func (d *Decoder) yuvImage(buf *bytes.Buffer, width, height int, subsample image.YCbCrSubsampleRatio) *image.YCbCr {
	rect := image.Rect(0, 0, width, height)
	data := buf.Bytes()

	switch subsample {
	case image.YCbCrSubsampleRatio420:
		ySize := width * height
		uvSize := (width / 2) * (height / 2)
		y0, y1 := 0, ySize
		u0, u1 := ySize, ySize+uvSize
		v0, v1 := ySize+uvSize, ySize+uvSize+uvSize
		return &image.YCbCr{
			Y:              data[y0:y1],
			Cb:             data[u0:u1],
			Cr:             data[v0:v1],
			YStride:        width,
			CStride:        width / 2,
			Rect:           rect,
			SubsampleRatio: subsample,
		}
	case image.YCbCrSubsampleRatio444:
		ySize := width * height
		uvSize := width * height
		y0, y1 := 0, ySize
		u0, u1 := ySize, ySize+uvSize
		v0, v1 := ySize+uvSize, ySize+uvSize+uvSize
		return &image.YCbCr{
			Y:              data[y0:y1],
			Cb:             data[u0:u1],
			Cr:             data[v0:v1],
			YStride:        width,
			CStride:        width,
			Rect:           rect,
			SubsampleRatio: subsample,
		}
	default:
		panic(fmt.Sprintf("unsupport yuv format: %d", subsample))
	}
}

func CreateDecoder(funcs ...decoderParameterFunc) (*Decoder, error) {
	decParam := defaultDecoderParameter()
	for _, fn := range funcs {
		fn(decParam)
	}

	api := unsafe.Pointer(C.decoder_api_get())
	param := unsafe.Pointer(C.decoder_parameters_create(
		(*C.xvc_decoder_api)(api),
	))
	defer func() {
		C.decoder_parameters_destroy(
			(*C.xvc_decoder_api)(api),
			(*C.xvc_decoder_parameters)(param),
		)
	}()

	ret := C.decoder_parameters_set_default(
		(*C.xvc_decoder_api)(api),
		(*C.xvc_decoder_parameters)(param),
	)
	if ret != C.XVC_DEC_OK {
		return nil, DecReturnCode(ret)
	}

	decParam.setCParam((*C.xvc_decoder_parameters)(param))

	if ret := C.decoder_parameters_check(
		(*C.xvc_decoder_api)(api),
		(*C.xvc_decoder_parameters)(param),
	); ret != C.XVC_DEC_OK {
		return nil, DecReturnCode(ret)
	}

	dec := unsafe.Pointer(C.decoder_create(
		(*C.xvc_decoder_api)(api),
		(*C.xvc_decoder_parameters)(param),
	))
	decoder := &Decoder{api, dec, decParam.bufferPoolFunc()}
	runtime.SetFinalizer(decoder, finalizeDecoder)
	return decoder, nil
}

func finalizeDecoder(decoder *Decoder) {
	DestroyDecoder(decoder)
}

func DestroyDecoder(decoder *Decoder) error {
	runtime.SetFinalizer(decoder, nil) // clear finalizer

	if ret := C.decoder_destroy(
		(*C.xvc_decoder_api)(decoder.api),
		(*C.xvc_decoder)(decoder.decoder),
	); ret != C.XVC_DEC_OK {
		return DecReturnCode(ret)
	}

	return nil
}
