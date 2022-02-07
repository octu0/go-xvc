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
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"runtime"
	"sync/atomic"
	"unsafe"
)

type NALUnit struct {
	buffer      *bytes.Buffer
	size        uint32
	nalUnitType uint32
	userData    int64
	closed      int32
	closeFunc   func()
}

func (n *NALUnit) Close() {
	if atomic.CompareAndSwapInt32(&n.closed, 0, 1) {
		n.closeFunc()
	}
}

func (n *NALUnit) Bytes() []byte {
	return n.buffer.Bytes()
}

func (n *NALUnit) UserData() int64 {
	return n.userData
}

func (n *NALUnit) Type() NALUnitType {
	return NALUnitType(n.nalUnitType)
}

func (n *NALUnit) CNALBytes() (*C.uchar, C.size_t) {
	b := n.Bytes()
	return (*C.uchar)(unsafe.Pointer(&b[0])), (C.size_t)(n.size)
}

func (n *NALUnit) CUserData() C.int64_t {
	return C.int64_t(n.userData)
}

type encoderParameterFunc func(*encoderParameter)
type encoderParameter struct {
	width             int
	height            int
	framerate         float32
	chromaFormat      ChromaFormat
	colorMatrix       ColorMatrix
	qp                int
	deblock           int // 0: disabled, 1: enabled, 2:low complexity
	lowDelay          int // 0: off, 1: on
	speedMode         int // 0: Placebo, 1: Slow, 2: Fast
	tuneMode          int // 0: Visual quality, 1: PSNR
	threads           int // -1: auto-detect,  0: disabled, 1+: number of threads
	bitDepth          uint32
	internalBitDepath uint32
	restrictMode      int
	bufferPoolFunc    func() BufferPool
}

func (e *encoderParameter) setCParam(param *C.xvc_encoder_parameters) {
	param.width = C.int(e.width)
	param.height = C.int(e.height)
	param.framerate = C.double(e.framerate)
	param.qp = C.int(e.qp)
	param.deblock = C.int(e.deblock)
	param.low_delay = C.int(e.lowDelay)
	param.speed_mode = C.int(e.speedMode)
	param.tune_mode = C.int(e.tuneMode)
	param.threads = C.int(e.threads)
	param.input_bitdepth = C.uint32_t(e.bitDepth)
	param.internal_bitdepth = C.uint32_t(e.internalBitDepath)
	param.restricted_mode = C.int(e.restrictMode)

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
}

func defaultEncoderParameter() *encoderParameter {
	return &encoderParameter{
		chromaFormat:      ChromaFormat420,
		colorMatrix:       ColorMatrixUnified,
		qp:                32,
		deblock:           1,  // enable
		lowDelay:          1,  // on
		speedMode:         2,  // fast
		threads:           -1, // auto
		bitDepth:          8,
		internalBitDepath: 8,
		restrictMode:      3, // baseline
		bufferPoolFunc: func() BufferPool {
			return newSimpleBufferPool(4 * 1024)
		},
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

func EncoderBufferPool(fn func() BufferPool) encoderParameterFunc {
	return func(p *encoderParameter) {
		p.bufferPoolFunc = fn
	}
}

type Encoder struct {
	api     unsafe.Pointer // xvc_encoder_api*
	encoder unsafe.Pointer // xvc_encoder*
	pool    BufferPool
}

func (e *Encoder) Encode(y, u, v []byte, strideY, strideU, strideV int, userData int64) ([]*NALUnit, error) {
	ret := unsafe.Pointer(C.encoder_encode2(
		(*C.xvc_encoder_api)(e.api),
		(*C.xvc_encoder)(e.encoder),
		(*C.uchar)(unsafe.Pointer(&y[0])),
		(*C.uchar)(unsafe.Pointer(&u[0])),
		(*C.uchar)(unsafe.Pointer(&v[0])),
		C.int(strideY),
		C.int(strideU),
		C.int(strideV),
		C.int64_t(userData),
	))
	if ret == nil {
		return nil, fmt.Errorf("encode2 not succeed")
	}
	result := (*C.encode_result_t)(ret)
	defer C.free_encode_result(result)

	return e.copyNALUnits(result)
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

	nalUnits, err := e.copyNALUnits(result)
	if err != nil {
		return nil, false
	}
	return nalUnits, true
}

func (e *Encoder) copyNALUnits(result *C.encode_result_t) ([]*NALUnit, error) {
	numNals := int(result.num_of_nals)
	nals := []C.encode_nal_unit_buf_t{}
	s := (*reflect.SliceHeader)(unsafe.Pointer(&nals))
	s.Cap = numNals
	s.Len = numNals
	s.Data = uintptr(unsafe.Pointer(result.nals))

	nalUnits := make([]*NALUnit, numNals)
	for i := 0; i < numNals; i += 1 {
		bufSize := uint32(nals[i].size) // size_t = unsigned long

		buf := e.pool.Get()
		// [0:4] size header
		// [4:]  nal data
		buf.Grow(4 + int(bufSize))

		nalSize := [4]int8{
			int8(nals[i].nal_size[0]),
			int8(nals[i].nal_size[1]),
			int8(nals[i].nal_size[2]),
			int8(nals[i].nal_size[3]),
		}
		for _, v := range nalSize {
			if err := binary.Write(buf, binary.BigEndian, v); err != nil {
				return nil, err
			}
		}
		buf.Write(C.GoBytes(unsafe.Pointer(nals[i].buf), C.int(bufSize)))

		nalUnits[i] = &NALUnit{
			buffer:      buf,
			size:        bufSize,
			nalUnitType: uint32(nals[i].nal_unit_type),
			userData:    int64(nals[i].user_data),
			closed:      int32(0),
			closeFunc: func() {
				buf.Reset()
				e.pool.Put(buf)
			},
		}
	}
	return nalUnits, nil
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
	encoder := &Encoder{api, enc, encParam.bufferPoolFunc()}
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
