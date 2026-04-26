#ifndef API_ROUTES_H
#define API_ROUTES_H

#include "db.c"
#include "gen_pdf.c"
#include "hiredis/adapters/libuv.h"
#include "libs/httpserver/httpserver.h"
#include "utils.c"

// POLLING

void poll_http(uv_timer_t *handle) {
  http_server_t *server = handle->data;
  while (http_server_poll(server) > 0)
    ;
}

#define API_V1_PREFIX "/api/v1"

#define JSON_ERROR_RESPONSE_SIZE 1024
#define JSON_CONTENT_TYPE "application/json"

#define GET_METHOD "GET"
#define POST_METHOD "POST"
#define PUT_METHOD "PUT"
#define DELETE_METHOD "DELETE"

// Check if the request target is the same as the target
int request_target_is(struct http_request_s *request, char *target);
// Check if the request method is the same as the method
int request_method_is(struct http_request_s *request, char *method);
// Check if the request target and method are the same as the target and method
int check_target_and_method(struct http_request_s *request, char *uri,
                            char *method);
// Create a JSON error response
char *create_json_error_response(int status, char const *message);
int send_json_error_response(struct http_request_s *request, int status,
                             char const *message);
int parse_invoice_create_request(http_string_t request_body, InvoiceModel *inv);

// MARK:- Handlers

void handle_health_request(struct http_request_s *request);
void handle_not_found_request(struct http_request_s *request);
void handle_create_invoice_request(struct http_request_s *request);
void handle_get_invoices_request(struct http_request_s *request);
void handle_gen_invoice_request(struct http_request_s *request);

void handle_request(struct http_request_s *request) {
  if (check_target_and_method(request, "/health", GET_METHOD)) {
    handle_health_request(request);
  } else if (check_target_and_method(request, "/invoices", POST_METHOD)) {
    handle_create_invoice_request(request);
  } else if (check_target_and_method(request, "/invoices", GET_METHOD)) {
    handle_get_invoices_request(request);
  } else if (check_target_and_method(request, "/gen-invoice", POST_METHOD)) {
    handle_gen_invoice_request(request);
  } else {
    handle_not_found_request(request);
  }
}

void handle_health_request(struct http_request_s *request) {
  char *body = "{\"message\": \"ok\"}";

  struct http_response_s *response = http_response_init();
  http_response_status(response, 200);
  http_response_header(response, "Content-Type", JSON_CONTENT_TYPE);
  http_response_body(response, body, strlen(body));
  http_respond(request, response);
}

void handle_create_invoice_request(struct http_request_s *request) {
  int rc = 1;

  http_string_t request_body = http_request_body(request);
  printf("request_body: %s\n", request_body.buf);

  InvoiceModel inv = invoice_model_init_empty();

  rc = parse_invoice_create_request(request_body, &inv);
  if (rc != 0) {
    send_json_error_response(request, 422, "Invalid request body");
    return;
  }

  rc = db_create_invoice(&inv);
  if (rc != 0) {
    send_json_error_response(request, 500, "Failed to create invoice");
    goto free_invoice_model_error;
  }
  char *body = json_invoice_model(inv);
  if (body == NULL) {
    send_json_error_response(request, 500,
                             "Failed to create invoice json response");
    goto free_body_error;
  }

  struct http_response_s *response = http_response_init();
  http_response_status(response, 204);
  http_response_header(response, "Content-Type", JSON_CONTENT_TYPE);
  http_response_body(response, body, strlen(body));
  http_respond(request, response);

free_body_error:
  free(body);

free_invoice_model_error:
  free_invoice_model(inv);
}

void handle_get_invoices_request(struct http_request_s *request) {
  int rc = 1;

  size_t count;
  InvoiceModel *invoices = db_get_all_invoices(&count);
  if (invoices == NULL) {
    send_json_error_response(request, 500, "Failed to get all invoices");
    goto free_invoices_error;
  }

  char *body = json_invoice_models(invoices, count);
  if (body == NULL) {
    send_json_error_response(request, 500,
                             "Failed to get all invoices json response");
    goto free_body_error;
  }

  struct http_response_s *response = http_response_init();
  http_response_status(response, 200);
  http_response_header(response, "Content-Type", JSON_CONTENT_TYPE);
  http_response_body(response, body, strlen(body));
  http_respond(request, response);

free_body_error:
  free(body);

free_invoices_error:
  free_invoice_models(invoices, count);
}

void handle_gen_invoice_request(struct http_request_s *request) {
  int rc = 1;

  http_string_t request_body = http_request_body(request);
  printf("request_body: %s\n", request_body.buf);

  InvoiceModel inv = invoice_model_init_empty();

  rc = parse_invoice_create_request(request_body, &inv);
  if (rc != 0) {
    send_json_error_response(request, 422, "Invalid request body");
    return;
  }

  rc = gen_invoice_pdf(&inv);
  if (rc != 0) {
    send_json_error_response(request, 500, "Failed to generate invoice pdf");
    goto free_invoice_model_error;
  }

  char *body = "{\"message\": \"Invoice pdf generated successfully\"}";
  struct http_response_s *response = http_response_init();
  http_response_status(response, 200);
  http_response_header(response, "Content-Type", JSON_CONTENT_TYPE);
  http_response_body(response, body, strlen(body));
  http_respond(request, response);

free_invoice_model_error:
  free_invoice_model(inv);
}

void handle_not_found_request(struct http_request_s *request) {
  char *body = create_json_error_response(404, "Not Found");

  struct http_response_s *response = http_response_init();
  http_response_status(response, 404);
  http_response_header(response, "Content-Type", JSON_CONTENT_TYPE);
  http_response_body(response, body, strlen(body));
  http_respond(request, response);

  free(body);
}

// MARK:- Utils

// Check if the request target is the same as the target
int request_target_is(struct http_request_s *request, char *target) {
  http_string_t url = http_request_target(request);
  int len = strlen(target);
  return len == url.len && memcmp(url.buf, target, url.len) == 0;
}

// Check if the request method is the same as the method
int request_method_is(struct http_request_s *request, char *method) {
  http_string_t method_str = http_request_method(request);
  int len = strlen(method);
  return len == method_str.len &&
         memcmp(method_str.buf, method, method_str.len) == 0;
}

// Check if the request target and method are the same as the target and method
int check_target_and_method(struct http_request_s *request, char *uri,
                            char *method) {
  char target[1024];
  cnstr(target, API_V1_PREFIX, uri);
  return request_target_is(request, target) &&
         request_method_is(request, method);
}

char *create_json_error_response(int status, char const *message) {
  char *response = calloc(JSON_ERROR_RESPONSE_SIZE, sizeof(char));
  // json object with error, status, timestamp
  // current time in RFC3339 format
  char timestamp[32];
  time_t now = time(NULL);
  strftime(timestamp, sizeof(timestamp), "%Y-%m-%dT%H:%M:%S%z",
           localtime(&now));
  snprintf(response, JSON_ERROR_RESPONSE_SIZE,
           "{\"error\": \"%s\", \"status\": %d, \"timestamp\": \"%s\"}",
           message, status, timestamp);
  return response;
}

int send_json_error_response(struct http_request_s *request, int status,
                             char const *message) {
  struct http_response_s *response = http_response_init();
  http_response_status(response, status);
  http_response_header(response, "Content-Type", JSON_CONTENT_TYPE);
  char *body = create_json_error_response(status, message);
  http_response_body(response, body, strlen(body));
  http_respond(request, response);
  free(body);
  return 0;
}

// TODO: Fix buffer overflows
int parse_invoice_create_request(http_string_t request_body,
                                 InvoiceModel *inv) {
  char user_id[10] = {0};
  get_json_value((char *)request_body.buf, "user_id", user_id);
  inv->user_id = atoi(user_id);
  if (inv->user_id <= 0) {
    return 1;
  }
  char order_sid[100] = {0};
  get_json_value((char *)request_body.buf, "order_sid", order_sid);
  if (strlen(order_sid) == 0) {
    return 2;
  }
  inv->order_sid = strdup(order_sid);
  char product_sid[100] = {0};
  get_json_value((char *)request_body.buf, "product_sid", product_sid);
  if (strlen(product_sid) == 0) {
    return 4;
  }
  inv->product_sid = strdup(product_sid);
  return 0;
}

#endif
