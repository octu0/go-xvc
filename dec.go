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
	"fmt"
	"image"
	"runtime"
	"unsafe"
)

type DecodedPicture struct {
	api unsafe.Pointer // xvc_decoder_api*
	pic unsafe.Pointer // xvc_decoded_picture*
}

func (n *DecodedPicture) Image() (NALUnitType, ColorMatrix, image.Image) {
	p := (*C.xvc_decoded_picture)(n.pic)
	nalType := NALUnitType(p.stats.nal_unit_type)
	colorMatrix := ColorMatrix(p.stats.color_matrix)

	width, height := int(p.stats.width), int(p.stats.height)

	chromaFormat := ChromaFormat(p.stats.chroma_format)
	switch chromaFormat {
	case ChromaFormat420:
		return nalType, colorMatrix, n.yuvImage(width, height, image.YCbCrSubsampleRatio420)
	case ChromaFormat444:
		return nalType, colorMatrix, n.yuvImage(width, height, image.YCbCrSubsampleRatio444)
	default:
		panic(fmt.Sprintf("unsupport format: %d", chromaFormat))
	}
}

func (n *DecodedPicture) UserData() int64 {
	p := (*C.xvc_decoded_picture)(n.pic)
	return int64(p.user_data)
}

func (n *DecodedPicture) yuvImage(width, height int, subsample image.YCbCrSubsampleRatio) *image.YCbCr {
	rect := image.Rect(0, 0, width, height)

	p := (*C.xvc_decoded_picture)(n.pic)
	planes := ([3]*C.char)(p.planes)
	strides := ([3]C.int)(p.stride)

	yStride := int(strides[0])
	uvStride := int(strides[1])

	switch subsample {
	case image.YCbCrSubsampleRatio420:
		ySize := height * yStride
		uvSize := (height / 2) * uvStride
		return &image.YCbCr{
			Y:              C.GoBytes(unsafe.Pointer(&planes[0]), C.int(ySize)),
			Cb:             C.GoBytes(unsafe.Pointer(&planes[1]), C.int(uvSize)),
			Cr:             C.GoBytes(unsafe.Pointer(&planes[2]), C.int(uvSize)),
			YStride:        yStride - width,
			CStride:        uvStride - (width / 2),
			Rect:           rect,
			SubsampleRatio: subsample,
		}
	case image.YCbCrSubsampleRatio444:
		ySize := height * yStride
		uvSize := height * uvStride
		return &image.YCbCr{
			Y:              C.GoBytes(unsafe.Pointer(&planes[0]), C.int(ySize)),
			Cb:             C.GoBytes(unsafe.Pointer(&planes[1]), C.int(uvSize)),
			Cr:             C.GoBytes(unsafe.Pointer(&planes[2]), C.int(uvSize)),
			YStride:        int(strides[0]) - width,
			CStride:        int(strides[1]) - width,
			Rect:           rect,
			SubsampleRatio: subsample,
		}
	default:
		panic(fmt.Sprintf("unsupport yuv format: %d", subsample))
	}
}

type decoderParameterFunc func(*decoderParameter)
type decoderParameter struct {
	width        int
	height       int
	chromaFormat ChromaFormat
	colorMatrix  ColorMatrix
	maxFramerate float32
	threads      int // -1: auto-detect
	bitDepth     int
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
		colorMatrix:  ColorMatrix709,
		threads:      -1, // auto
		bitDepth:     8,  // 8bit
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

type Decoder struct {
	api     unsafe.Pointer // xvc_decoder_api*
	decoder unsafe.Pointer // xvc_decoder*
}

func (d *Decoder) Decode(bytes []byte) error {
	ret := C.decoder_decode_nal(
		(*C.xvc_decoder_api)(d.api),
		(*C.xvc_decoder)(d.decoder),
		(*C.uchar)(unsafe.Pointer(&bytes[0])),
		C.size_t(len(bytes)),
		C.int64_t(0),
	)
	if ret != C.XVC_DEC_OK {
		return DecReturnCode(ret)
	}
	return nil
}

func (d *Decoder) DecodeNAL(nal *NALUnit) error {
	bytes, size := nal.CNALBytes()
	userData := nal.CUserData()

	ret := C.decoder_decode_nal(
		(*C.xvc_decoder_api)(d.api),
		(*C.xvc_decoder)(d.decoder),
		bytes,
		size,
		userData,
	)
	if ret != C.XVC_DEC_OK {
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
	ret := C.decoder_get_picture(
		(*C.xvc_decoder_api)(d.api),
		(*C.xvc_decoder)(d.decoder),
		pic,
	)
	if ret != C.XVC_DEC_OK {
		C.decoder_picture_destroy(
			(*C.xvc_decoder_api)(d.api),
			pic,
		)
		return nil, DecReturnCode(ret)
	}

	dpic := &DecodedPicture{d.api, unsafe.Pointer(pic)}
	runtime.SetFinalizer(dpic, finalizeDecodedPicture)
	return dpic, nil
}

func finalizeDecodedPicture(dpic *DecodedPicture) {
	DestroyDecodedPicture(dpic)
}

func DestroyDecodedPicture(dpic *DecodedPicture) error {
	runtime.SetFinalizer(dpic, nil) // clear finalizer

	if ret := C.decoder_picture_destroy(
		(*C.xvc_decoder_api)(dpic.api),
		(*C.xvc_decoded_picture)(dpic.pic),
	); ret != C.XVC_DEC_OK {
		return DecReturnCode(ret)
	}

	return nil
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
	decoder := &Decoder{api, dec}
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
