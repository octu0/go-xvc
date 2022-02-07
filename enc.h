#include <stdio.h>
#include <string.h>
#include "xvcenc.h"

#ifndef H_GO_XVC_ENC
#define H_GO_XVC_ENC

typedef struct encode_nal_unit_buf_t {
  unsigned char *buf;
  char nal_size[4];
  size_t size;
  uint32_t nal_unit_type;
  int64_t user_data;
} encode_nal_unit_buf_t;

typedef struct encode_result_t {
  encode_nal_unit_buf_t *nals;
  int num_of_nals;
} encode_result_t;

void free_encode_result(encode_result_t* result) {
  if(result != NULL) {
    if(result->nals != NULL) {
      for(int i = 0; i < result->num_of_nals; i += 1) {
        free(result->nals[i].buf);
        result->nals[i].buf = NULL;
      }
    }
    free(result->nals);
    result->nals = NULL;
    result->num_of_nals = 0;
  }
  free(result);
}

encode_result_t* create_encode_result(xvc_enc_nal_unit *nal_units, int num_nal_units) {
  encode_result_t* result = (encode_result_t *) malloc(sizeof(encode_result_t));
  if(result == NULL) {
    free_encode_result(result);
    return NULL;
  }
  memset(result, 0, sizeof(encode_result_t));
  result->nals = NULL;
  result->num_of_nals = 0;

  result->nals = (encode_nal_unit_buf_t *) malloc(num_nal_units * sizeof(encode_nal_unit_buf_t));
  if(result->nals == NULL) {
    free_encode_result(result);
    return NULL;
  }
  memset(result->nals, 0, num_nal_units * sizeof(encode_nal_unit_buf_t));

  for(int i = 0; i < num_nal_units; i += 1) {
    result->nals[i].buf = (unsigned char *) malloc(nal_units[i].size);
    if(result->nals[i].buf == NULL) {
      free_encode_result(result);
      return NULL;
    }
    result->nals[i].nal_size[0] = (nal_units[i].size) & 0xFF;
    result->nals[i].nal_size[1] = (nal_units[i].size >> 8) & 0xFF;
    result->nals[i].nal_size[2] = (nal_units[i].size >> 16) & 0xFF;
    result->nals[i].nal_size[3] = (nal_units[i].size >> 24) & 0xFF;
    result->nals[i].size = nal_units[i].size;
    result->nals[i].nal_unit_type = nal_units[i].stats.nal_unit_type;
    result->nals[i].user_data = nal_units[i].user_data;
    memcpy(result->nals[i].buf, nal_units[i].bytes, nal_units[i].size);
  }
  result->num_of_nals = num_nal_units;

  return result;
}

const xvc_encoder_api* encoder_api_get() {
  return xvc_encoder_api_get();
}

xvc_encoder_parameters* encoder_parameters_create(xvc_encoder_api* api) {
  return (api->parameters_create)();
}

xvc_enc_return_code encoder_parameters_set_default(xvc_encoder_api* api, xvc_encoder_parameters* param) {
  return (api->parameters_set_default)(param);
}

xvc_enc_return_code encoder_parameters_check(xvc_encoder_api* api, xvc_encoder_parameters* param) {
  return (api->parameters_check)(param);
}

xvc_enc_return_code encoder_parameters_destroy(xvc_encoder_api* api, xvc_encoder_parameters* param) {
  return (api->parameters_destroy)(param);
}

xvc_encoder* encoder_create(xvc_encoder_api* api, xvc_encoder_parameters* param) {
  return (api->encoder_create)(param);
}

xvc_enc_return_code encoder_destroy(xvc_encoder_api* api, xvc_encoder* encoder) {
  return (api->encoder_destroy)(encoder);
}

encode_result_t* encoder_encode2(
  xvc_encoder_api* api,
  xvc_encoder* encoder,
  const unsigned char *y_plane,
  const unsigned char *u_plane,
  const unsigned char *v_plane,
  int y_stride,
  int u_stride,
  int v_stride,
  int64_t user_data
) {
  const unsigned char *plane_bytes[3] = {y_plane, u_plane, v_plane};
  int plane_stride[3] = {y_stride, u_stride, v_stride};

  xvc_enc_nal_unit *nal_units;
  int num_nal_units;
  xvc_enc_return_code ret = (api->encoder_encode2)(encoder, plane_bytes, plane_stride, &nal_units, &num_nal_units, NULL, user_data);
  if(ret == XVC_ENC_OK) {
    return create_encode_result(nal_units, num_nal_units);
  }
  return NULL;
}

encode_result_t* encoder_flush(
  xvc_encoder_api* api,
  xvc_encoder* encoder
) {
  xvc_enc_nal_unit *nal_units;
  int num_nal_units;
  xvc_enc_return_code ret = (api->encoder_flush)(encoder, &nal_units, &num_nal_units, NULL);
  if(ret == XVC_ENC_OK || ret == XVC_ENC_NO_MORE_OUTPUT) {
    return create_encode_result(nal_units, num_nal_units);
  }
  return NULL;
}

#endif
