package xvc

type EncReturnCode uint8

const (
	EncOK                        EncReturnCode = 0
	EncNoMoreOutput                            = 1
	EncInvalidArgument                         = 10
	EncInvalidParameter                        = 20
	EncSizeTooSmall                            = 21
	EncUnsupportedChromaFormat                 = 22
	EncBitDepthOutOfRange                      = 23
	EncCompiledBitDepthTooLow                  = 24
	EncFramerateOutOfRange                     = 25
	EncQPOutOfRange                            = 26
	EncSubGOPLengthTooLarge                    = 27
	EncDeblockingSettingsInvalid               = 28
	EncTooManyRefPics                          = 29
	EncSizeTooLarge                            = 30
	EncNoSuchPreset                            = 100
)

func (c EncReturnCode) Error() string {
	switch c {
	case EncOK:
		return "XVC_ENC_OK"
	case EncNoMoreOutput:
		return "XVC_ENC_NO_MORE_OUTPUT"
	case EncInvalidArgument:
		return "XVC_ENC_INVALID_ARGUMENT"
	case EncInvalidParameter:
		return "XVC_ENC_INVALID_PARAMETER"
	case EncSizeTooSmall:
		return "XVC_ENC_SIZE_TOO_SMALL"
	case EncUnsupportedChromaFormat:
		return "XVC_ENC_UNSUPPORTED_CHROMA_FORMAT"
	case EncBitDepthOutOfRange:
		return "XVC_ENC_BITDEPTH_OUT_OF_RANGE"
	case EncCompiledBitDepthTooLow:
		return "XVC_ENC_COMPILED_BITDEPTH_TOO_LOW"
	case EncFramerateOutOfRange:
		return "XVC_ENC_FRAMERATE_OUT_OF_RANGE"
	case EncQPOutOfRange:
		return "XVC_ENC_QP_OUT_OF_RANGE"
	case EncSubGOPLengthTooLarge:
		return "XVC_ENC_SUB_GOP_LENGTH_TOO_LARGE"
	case EncDeblockingSettingsInvalid:
		return "XVC_ENC_DEBLOCKING_SETTINGS_INVALID"
	case EncTooManyRefPics:
		return "XVC_ENC_TOO_MANY_REF_PICS"
	case EncSizeTooLarge:
		return "XVC_ENC_SIZE_TOO_LARGE"
	case EncNoSuchPreset:
		return "XVC_ENC_NO_SUCH_PRESET"
	}
	return "unknown error"
}

type DecReturnCode uint8

const (
	DecOK                                          DecReturnCode = 0
	DecNoDecodedPic                                              = 1
	DecNotConfirming                                             = 10
	DecInvalidArgument                                           = 20
	DecInvalidParameter                                          = 30
	DecFramerateOutOfRange                                       = 31
	DecBitDepthOutOfRange                                        = 32
	DecBitstreamVersionHigherThanDecoder                         = 33
	DecNoSegmentHeaderDecoded                                    = 34
	DecBitstreamDepthTooHigh                                     = 35
	DecBitstreamVersionLowerThanSupportedByDecoder               = 36
)

func (c DecReturnCode) Error() string {
	switch c {
	case DecOK:
		return "XVC_DEC_OK"
	case DecNoDecodedPic:
		return "XVC_DEC_NO_DECODED_PIC"
	case DecNotConfirming:
		return "XVC_DEC_NOT_CONFORMING"
	case DecInvalidArgument:
		return "XVC_DEC_INVALID_ARGUMENT"
	case DecInvalidParameter:
		return "XVC_DEC_INVALID_PARAMETER"
	case DecFramerateOutOfRange:
		return "XVC_DEC_FRAMERATE_OUT_OF_RANGE"
	case DecBitDepthOutOfRange:
		return "XVC_DEC_BITDEPTH_OUT_OF_RANGE"
	case DecBitstreamVersionHigherThanDecoder:
		return "XVC_DEC_BITSTREAM_VERSION_HIGHER_THAN_DECODER"
	case DecNoSegmentHeaderDecoded:
		return "XVC_DEC_NO_SEGMENT_HEADER_DECODED"
	case DecBitstreamDepthTooHigh:
		return "XVC_DEC_BITSTREAM_BITDEPTH_TOO_HIGH"
	case DecBitstreamVersionLowerThanSupportedByDecoder:
		return "XVC_DEC_BITSTREAM_VERSION_LOWER_THAN_SUPPORTED_BY_DECODER"
	}
	return "unknown error"
}

type ChromaFormat uint8

const (
	ChromaFormatMonochrome ChromaFormat = 0
	ChromaFormat420                     = 1
	ChromaFormat422                     = 2
	ChromaFormat444                     = 3
	ChromaFormatARGB                    = 4
	ChromaFormatUnified                 = 255
)

func (f ChromaFormat) String() string {
	switch f {
	case ChromaFormatMonochrome:
		return "monochrome"
	case ChromaFormat420:
		return "420"
	case ChromaFormat422:
		return "422"
	case ChromaFormat444:
		return "444"
	case ChromaFormatARGB:
		return "argb"
	case ChromaFormatUnified:
		return "unified"
	}
	return "unknown chroma_format"
}

type ColorMatrix uint8

const (
	ColorMatrixUnified ColorMatrix = 0
	ColorMatrix601                 = 1
	ColorMatrix709                 = 2
	ColorMatrix2020                = 3
)

func (m ColorMatrix) String() string {
	switch m {
	case ColorMatrixUnified:
		return "unified"
	case ColorMatrix601:
		return "601"
	case ColorMatrix709:
		return "709"
	case ColorMatrix2020:
		return "2020"
	}
	return "unknown color_matrix"
}

type NALUnitType uint8

const (
	IntraPicture             NALUnitType = 0
	IntraAccessPicture                   = 1
	PredictedPicture                     = 2
	PredictedAccessPicture               = 3
	BipredictedPicture                   = 4
	BipredictedAccessPicture             = 5
	ReservedPictureType6                 = 6
	ReservedPictureType7                 = 7
	ReservedPictureType8                 = 8
	ReservedPictureType9                 = 9
	ReservedPictureType10                = 10
	SegmentHeader                        = 16
	Sei                                  = 17
	AccessUnitDelimiter                  = 18
	EndOfSegment                         = 19
)

func (t NALUnitType) String() string {
	switch t {
	case IntraPicture:
		return "intra_picture"
	case IntraAccessPicture:
		return "intra_access_picture"
	case PredictedPicture:
		return "predicted_picture"
	case PredictedAccessPicture:
		return "predicted_access_picture"
	case BipredictedPicture:
		return "bipredicted_picture"
	case BipredictedAccessPicture:
		return "bipredicted_access_picture"
	case ReservedPictureType6:
		return "reserved_picture_type6"
	case ReservedPictureType7:
		return "reserved_picture_type7"
	case ReservedPictureType8:
		return "reserved_picture_type8"
	case ReservedPictureType9:
		return "reserved_picture_type9"
	case ReservedPictureType10:
		return "reserved_picture_type10"
	case SegmentHeader:
		return "segment_header"
	case Sei:
		return "sei"
	case AccessUnitDelimiter:
		return "access_unit_delimiter"
	case EndOfSegment:
		return "end_of_segment"
	}
	return "unknown_nal"
}
