#include <stdio.h>
#include <string.h>
#include "xvcdec.h"

#ifndef H_GO_XVC_DEC
#define H_GO_XVC_DEC

const xvc_decoder_api* decoder_api_get() {
  return xvc_decoder_api_get();
}

xvc_decoder_parameters* decoder_parameters_create(xvc_decoder_api* api) {
  return (api->parameters_create)();
}

xvc_dec_return_code decoder_parameters_set_default(xvc_decoder_api* api, xvc_decoder_parameters* param) {
  return (api->parameters_set_default)(param);
}

xvc_dec_return_code decoder_parameters_check(xvc_decoder_api* api, xvc_decoder_parameters* param) {
  return (api->parameters_check)(param);
}

xvc_dec_return_code decoder_parameters_destroy(xvc_decoder_api* api, xvc_decoder_parameters* param) {
  return (api->parameters_destroy)(param);
}

xvc_decoder* decoder_create(xvc_decoder_api* api, xvc_decoder_parameters* param) {
  return (api->decoder_create)(param);
}

xvc_dec_return_code decoder_destroy(xvc_decoder_api* api, xvc_decoder* decoder) {
  return (api->decoder_destroy)(decoder);
}

xvc_decoded_picture* decoder_picture_create(xvc_decoder_api* api, xvc_decoder* decoder) {
  return (api->picture_create)(decoder);
}

xvc_dec_return_code decoder_picture_destroy(xvc_decoder_api* api, xvc_decoded_picture* pic) {
  return (api->picture_destroy)(pic);
}

xvc_dec_return_code decoder_decode_nal(
  xvc_decoder_api* api,
  xvc_decoder* decoder,
  const unsigned char *nal_unit,
  size_t nal_unit_size,
  int64_t user_data
) {
  return (api->decoder_decode_nal)(decoder, nal_unit, nal_unit_size, user_data);
}

xvc_dec_return_code decoder_get_picture(
  xvc_decoder_api* api,
  xvc_decoder* decoder,
  xvc_decoded_picture* out_pic
) {
  return (api->decoder_get_picture)(decoder, out_pic);
}

xvc_dec_return_code decoder_flush(xvc_decoder_api* api, xvc_decoder* decoder) {
  return (api->decoder_flush)(decoder);
}

#endif
