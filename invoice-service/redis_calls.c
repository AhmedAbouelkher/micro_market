#ifndef REDIS_CALLS_H
#define REDIS_CALLS_H

#include "db.c"
#include "gen_pdf.c"
#include "hiredis/async.h"
#include "hiredis/hiredis.h"
#include "utils.c"

#define DEFAULT_REDIS_CHANNEL "create_invoice"

void invoiceGenCallback(redisAsyncContext *c, void *r, void *privdata);

void connectCallback(const redisAsyncContext *c, int status) {
  if (status != REDIS_OK) {
    fprintf(stderr, "failed to connect to Redis: %s\n", c->errstr);
    return;
  }
  fprintf(stdout, "connected to Redis\n");

  // get the channel name from the environment variable
  const char *channel_name = get_env("REDIS_CHANNEL", DEFAULT_REDIS_CHANNEL);
  if (channel_name == NULL) {
    fprintf(stderr, "REDIS_CHANNEL environment variable is not set\n");
    return;
  }

  if (redisAsyncCommand(c, invoiceGenCallback, (void *)channel_name,
                        "SUBSCRIBE %s", channel_name) != REDIS_OK) {
    fprintf(stderr, "failed to subscribe to Redis channel: %s\n", c->errstr);
    redisAsyncFree(c);
  }
}

void disconnectCallback(const redisAsyncContext *c, int status) {
  if (status != REDIS_OK) {
    fprintf(stderr, "failed to disconnect from Redis: %s\n", c->errstr);
    return;
  }

  printf("Disconnected from Redis\n");
}

// MARK:- Handlers

// TODO: Fix buffer overflows
int parse_redis_invoice_gen(char *data, InvoiceModel *inv) {
  char user_id[10] = {0};
  get_json_value(data, "user_id", user_id);
  inv->user_id = atoi(user_id);
  if (inv->user_id <= 0) {
    return 1;
  }

  char order_sid[100] = {0};
  get_json_value(data, "order_sid", order_sid);
  if (strlen(order_sid) == 0)
    return 2;
  inv->order_sid = strdup(order_sid);

  char product_sid[100] = {0};
  get_json_value(data, "product_sid", product_sid);
  if (strlen(product_sid) == 0)
    return 3;
  inv->product_sid = strdup(product_sid);

  char user_name[100] = {0};
  get_json_value(data, "user_name", user_name);
  if (strlen(user_name) == 0)
    return 4;
  inv->user_name = strdup(user_name);

  char user_email[100] = {0};
  get_json_value(data, "user_email", user_email);
  if (strlen(user_email) == 0)
    return 5;
  inv->user_email = strdup(user_email);

  char product_name[100] = {0};
  get_json_value(data, "product_name", product_name);
  if (strlen(product_name) == 0)
    return 6;
  inv->product_name = strdup(product_name);

  char product_price[10] = {0};
  get_json_value(data, "product_price", product_price);
  inv->price_per_item = atoi(product_price);
  if (inv->price_per_item <= 0)
    return 7;

  char quantity[10] = {0};
  get_json_value(data, "quantity", quantity);
  inv->quantity = atoi(quantity);
  if (inv->quantity <= 0)
    return 8;

  char total[10] = {0};
  get_json_value(data, "total", total);
  inv->total = atoi(total);
  if (inv->total <= 0)
    return 9;
  return 0;
}

void invoiceGenCallback(redisAsyncContext *c, void *r, void *privdata) {
  redisReply *reply = r;
  if (reply == NULL)
    return;

  if (reply->type != REDIS_REPLY_ARRAY)
    return;

  size_t elements = reply->elements;
  if (elements < 3)
    return;

  if (strcmp(reply->element[0]->str, "message") != 0 ||
      strcmp(reply->element[1]->str, (char *)privdata) != 0)
    return;

  int rc = 1;

  char *data = reply->element[2]->str;
  printf("Received invoice generation request: %s\n", data);

  InvoiceModel inv = invoice_model_init_empty();
  rc = parse_redis_invoice_gen(data, &inv);
  if (rc != 0) {
    fprintf(stderr, "failed to parse invoice generation request: %d\n", rc);
    goto rollback_invoice_creation;
  }

  rc = db_create_invoice(&inv);
  if (rc != 0) {
    fprintf(stderr, "failed to create invoice: %d\n", rc);
    goto rollback_invoice_creation;
  }

  rc = gen_invoice_pdf(&inv);
  if (rc != 0) {
    fprintf(stderr, "failed to generate invoice pdf: %d\n", rc);
    goto free_invoice_model_and_exit;
  }

rollback_invoice_creation:
  if (inv.id > 0) {
    rc = db_delete_invoice(inv.id);
    if (rc != 0)
      fprintf(stderr, "failed to delete invoice: %d\n", rc);
  }

free_invoice_model_and_exit:
  free_invoice_model(inv);
}

#endif