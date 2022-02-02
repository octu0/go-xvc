package xvc

/*
#cgo CFLAGS: -I${SRCDIR}/include -I/usr/local/include -I/usr/include
#cgo darwin LDFLAGS: -L${SRCDIR} -L/usr/local/lib -L/usr/lib -lxvcenc -lc++
#cgo linux  LDFLAGS: -L${SRCDIR} -L/usr/local/lib -L/usr/lib -lxvcenc -lc++
#include <stdint.h>
#include <stdlib.h>

#include "enc.h"
*/
import "C"

import (
	"fmt"
	"reflect"
	"runtime"
	"time"
	"unsafe"
)

type NALUnit struct {
	bytes       []byte
	size        uint32
	nalUnitType uint32
	userData    int64
}

func (n *NALUnit) Bytes() []byte {
	return n.bytes
}

func (n *NALUnit) UserData() int64 {
	return n.userData
}

func (n *NALUnit) Type() NALUnitType {
	return NALUnitType(n.nalUnitType)
}

func (n *NALUnit) CNALBytes() (*C.uchar, C.size_t) {
	return (*C.uchar)(unsafe.Pointer(&n.bytes[0])), (C.size_t)(n.size)
}

func (n *NALUnit) CUserData() C.int64_t {
	return C.int64_t(n.userData)
}

type encoderParameterFunc func(*encoderParameter)
type encoderParameter struct {
	width        int
	height       int
	framerate    float32
	chromaFormat ChromaFormat
	colorMatrix  ColorMatrix
	qp           int
	lowDelay     int // 0: off, 1: on
	speedMode    int // 0: Placebo, 1: Slow, 2: Fast
	tuneMode     int // 0: Visual quality, 1: PSNR
	threads      int // -1: auto-detect,  0: disabled, 1+: number of threads
	restrictMode int
}

func (e *encoderParameter) setCParam(param *C.xvc_encoder_parameters) {
	param.width = C.int(e.width)
	param.height = C.int(e.height)
	param.framerate = C.double(e.framerate)

	switch e.chromaFormat {
	case ChromaFormatMonochrome:
		param.chroma_format = C.XVC_ENC_CHROMA_FORMAT_MONOCHROME
	case ChromaFormat420:
		param.chroma_format = C.XVC_ENC_CHROMA_FORMAT_420
	case ChromaFormat422:
		param.chroma_format = C.XVC_ENC_CHROMA_FORMAT_422
	case ChromaFormat444:
		param.chroma_format = C.XVC_ENC_CHROMA_FORMAT_444
	case ChromaFormatUnified:
		param.chroma_format = C.XVC_ENC_CHROMA_FORMAT_UNDEFINED
	}

	switch e.colorMatrix {
	case ColorMatrix601:
		param.color_matrix = C.XVC_ENC_COLOR_MATRIX_601
	case ColorMatrix709:
		param.color_matrix = C.XVC_ENC_COLOR_MATRIX_709
	case ColorMatrix2020:
		param.color_matrix = C.XVC_ENC_COLOR_MATRIX_2020
	case ColorMatrixUnified:
		param.color_matrix = C.XVC_ENC_COLOR_MATRIX_UNDEFINED
	}

	param.qp = C.int(e.qp)
	param.low_delay = C.int(e.lowDelay)
	param.speed_mode = C.int(e.speedMode)
	param.tune_mode = C.int(e.tuneMode)
	param.threads = C.int(e.threads)
	param.restricted_mode = C.int(e.restrictMode)
}

func defaultEncoderParameter() *encoderParameter {
	return &encoderParameter{
		chromaFormat: ChromaFormat420,
		colorMatrix:  ColorMatrix2020,
		qp:           32, // default
		lowDelay:     1,  // on
		speedMode:    2,  // fast
		threads:      -1, // auto
		restrictMode: 3,  // baseline
	}
}

func EncoderParameterWidth(width int) encoderParameterFunc {
	return func(p *encoderParameter) {
		p.width = width
	}
}

func EncoderParameterHeight(height int) encoderParameterFunc {
	return func(p *encoderParameter) {
		p.height = height
	}
}

func EncoderParameterFramerate(rate float32) encoderParameterFunc {
	return func(p *encoderParameter) {
		p.framerate = rate
	}
}

type Encoder struct {
	api     unsafe.Pointer // xvc_encoder_api*
	encoder unsafe.Pointer // xvc_encoder*
}

func (e *Encoder) Encode(y, u, v []byte, strideY, strideU, strideV int, t time.Time) ([]*NALUnit, error) {
	ret := unsafe.Pointer(C.encoder_encode2(
		(*C.xvc_encoder_api)(e.api),
		(*C.xvc_encoder)(e.encoder),
		(*C.uchar)(unsafe.Pointer(&y[0])),
		(*C.uchar)(unsafe.Pointer(&u[0])),
		(*C.uchar)(unsafe.Pointer(&v[0])),
		C.int(strideY),
		C.int(strideU),
		C.int(strideV),
		C.int64_t(t.UnixNano()),
	))
	if ret == nil {
		return nil, fmt.Errorf("encode2 not succeed")
	}
	result := (*C.encode_result_t)(ret)
	defer C.free_encode_result(result)

	return e.copyNALUnits(result), nil
}

func (e *Encoder) Flush() ([]*NALUnit, bool) {
	ret := unsafe.Pointer(C.encoder_flush(
		(*C.xvc_encoder_api)(e.api),
		(*C.xvc_encoder)(e.encoder),
	))
	if ret == nil {
		return nil, false
	}
	result := (*C.encode_result_t)(ret)
	defer C.free_encode_result(result)

	return e.copyNALUnits(result), true
}

func (e *Encoder) copyNALUnits(result *C.encode_result_t) []*NALUnit {
	numNals := int(result.num_of_nals)
	nals := []C.encode_nal_unit_buf_t{}
	s := (*reflect.SliceHeader)(unsafe.Pointer(&nals))
	s.Cap = numNals
	s.Len = numNals
	s.Data = uintptr(unsafe.Pointer(result.nals))

	nalUnits := make([]*NALUnit, numNals)
	for i := 0; i < numNals; i += 1 {
		bufSize := uint32(nals[i].size) // size_t = unsigned long
		buffer := make([]byte, bufSize) // todo pool
		copy(buffer, C.GoBytes(unsafe.Pointer(nals[i].buf), C.int(bufSize)))

		nalUnits[i] = &NALUnit{
			bytes:       buffer,
			size:        bufSize,
			nalUnitType: uint32(nals[i].nal_unit_type),
			userData:    int64(nals[i].user_data),
		}
	}
	return nalUnits
}

func CreateEncoder(funcs ...encoderParameterFunc) (*Encoder, error) {
	encParam := defaultEncoderParameter()
	for _, fn := range funcs {
		fn(encParam)
	}

	api := unsafe.Pointer(C.encoder_api_get())
	param := unsafe.Pointer(C.encoder_parameters_create(
		(*C.xvc_encoder_api)(api),
	))
	defer func() {
		C.encoder_parameters_destroy(
			(*C.xvc_encoder_api)(api),
			(*C.xvc_encoder_parameters)(param),
		)
	}()

	if ret := C.encoder_parameters_set_default(
		(*C.xvc_encoder_api)(api),
		(*C.xvc_encoder_parameters)(param),
	); ret != C.XVC_ENC_OK {
		return nil, EncReturnCode(ret)
	}

	encParam.setCParam((*C.xvc_encoder_parameters)(param))

	if ret := C.encoder_parameters_check(
		(*C.xvc_encoder_api)(api),
		(*C.xvc_encoder_parameters)(param),
	); ret != C.XVC_ENC_OK {
		return nil, EncReturnCode(ret)
	}

	enc := unsafe.Pointer(C.encoder_create(
		(*C.xvc_encoder_api)(api),
		(*C.xvc_encoder_parameters)(param),
	))
	encoder := &Encoder{api, enc}
	runtime.SetFinalizer(encoder, finalizeEncoder)
	return encoder, nil
}

func finalizeEncoder(encoder *Encoder) {
	DestroyEncoder(encoder)
}

func DestroyEncoder(encoder *Encoder) error {
	runtime.SetFinalizer(encoder, nil) // clear finalizer

	if ret := C.encoder_destroy(
		(*C.xvc_encoder_api)(encoder.api),
		(*C.xvc_encoder)(encoder.encoder),
	); ret != C.XVC_ENC_OK {
		return EncReturnCode(ret)
	}

	return nil
}
