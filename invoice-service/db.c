#ifndef DB_H
#define DB_H

#include "libs/sqlite3/sqlite3.c"
#include "libs/ulid/ulid.h"
#include "utils.c"
#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

sqlite3 *db;
struct ulid_generator ulid_generator;

int close_db() {
  if (db) {
    sqlite3_close(db);
    db = NULL;
  }
  return 0;
}

int run_migrations();

int open_db(const char *db_path) {
  close_db(); // just to be sure

  int rc = sqlite3_open(db_path, &db);
  if (rc != SQLITE_OK) {
    fprintf(stderr, "Error opening database: %s\n", sqlite3_errmsg(db));
    return 1;
  }

  if (ulid_generator_init(&ulid_generator, ULID_RELAXED) != 0) {
    fprintf(stderr, "Error initializing ulid generator: %s\n",
            sqlite3_errmsg(db));
    return 1;
  }

  rc = run_migrations();
  if (rc != 0) {
    fprintf(stderr, "Error running migrations: %s\n", sqlite3_errmsg(db));
    return 1;
  }

  return 0;
}

// MARK:- Models
#define INVOICE_TABLE_NAME "invoice_models"
#define MAX_QUERY_LENGTH 1024
#define MAX_CREATED_AT_LENGTH 100
#define JSON_INVOICE_MODEL_SIZE 1024

typedef struct {
  uint8_t id;
  char *sid;
  uint8_t user_id;
  char *order_sid;
  char *product_sid;
  time_t created_at;

  char *user_name;        // not for DB, used for PDF generation
  char *user_email;       // not for DB, used for PDF generation
  char *product_name;     // not for DB, used for PDF generation
  uint8_t price_per_item; // not for DB, used for PDF generation
  uint8_t quantity;       // not for DB, used for PDF generation
  uint8_t total;          // not for DB, used for PDF generation
} InvoiceModel;

InvoiceModel invoice_model_init(uint8_t user_id, char *order_sid,
                                char *product_sid) {
  unsigned char bin[16];
  char ulid[27];
  ulid_generate(&ulid_generator, ulid);
  ulid_decode(bin, ulid);
  char *sid = strdup(ulid);
  InvoiceModel inv = {
      .id = 0,
      .sid = sid,
      .user_id = user_id,
      .created_at = get_local_time_seconds_since_epoch(),
  };
  if (order_sid) {
    inv.order_sid = strdup(order_sid);
  }
  if (product_sid) {
    inv.product_sid = strdup(product_sid);
  }
  return inv;
}

InvoiceModel invoice_model_init_empty() {
  return invoice_model_init(0, NULL, NULL);
}

void free_invoice_model(InvoiceModel inv) {
  free(inv.sid);
  free(inv.order_sid);
  free(inv.product_sid);
  free(inv.user_name);
  free(inv.user_email);
  free(inv.product_name);
}

void free_invoice_models(InvoiceModel *invoices, size_t count) {
  for (size_t i = 0; i < count; i++) {
    free_invoice_model(invoices[i]);
  }
  free(invoices);
}

char *json_invoice_model(InvoiceModel inv) {
  char json[JSON_INVOICE_MODEL_SIZE] = {0};
  snprintf(json, JSON_INVOICE_MODEL_SIZE,
           "{\"id\": %u, \"sid\": \"%s\", \"user_id\": %u, \"order_sid\": "
           "\"%s\", \"product_sid\": \"%s\", \"created_at\": %ld}",
           inv.id, inv.sid, inv.user_id, inv.order_sid, inv.product_sid,
           inv.created_at);
  return strdup(json);
}

char *json_invoice_models(InvoiceModel *invoices, size_t count) {
  char *json = calloc(JSON_INVOICE_MODEL_SIZE * count + 2, sizeof(char));
  if (json == NULL) {
    return NULL;
  }
  strncpy(json, "[", 1);
  for (size_t i = 0; i < count; i++) {
    strncat(json, json_invoice_model(invoices[i]),
            strlen(json_invoice_model(invoices[i])));
    if (i < count - 1) {
      strncat(json, ",", 1);
    }
  }
  strncat(json, "]", 1);
  return json;
}

// MARK:- Migrations

// create table if not exists invoices
int create_invoices_table() {
  int rc = 1;

  char query[MAX_QUERY_LENGTH];
  snprintf(query, sizeof(query),
           "CREATE TABLE IF NOT EXISTS " INVOICE_TABLE_NAME " ("
           "id INTEGER PRIMARY KEY AUTOINCREMENT, "
           "sid TEXT, "
           "user_id INTEGER, "
           "order_sid TEXT, "
           "product_sid TEXT, "
           "created_at INTEGER"
           ")");

  rc = sqlite3_exec(db, query, NULL, NULL, NULL);
  if (rc != SQLITE_OK) {
    fprintf(stderr, "Error creating invoices table: %s\n", sqlite3_errmsg(db));
    goto free_and_exit;
  }

  // reset the query string
  // Only need one method to reset. memset clears all bytes to zero.
  memset(query, 0, sizeof(query));

  // create unique index on sid (if it does not exist)
  snprintf(query, sizeof(query),
           "CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_sid "
           "ON " INVOICE_TABLE_NAME " (sid)");
  rc = sqlite3_exec(db, query, NULL, NULL, NULL);
  if (rc != SQLITE_OK) {
    fprintf(stderr, "Error creating unique index on sid: %s\n",
            sqlite3_errmsg(db));
    goto free_and_exit;
  }

free_and_exit:
  return rc;
}

int run_migrations() {
  int rc = 1;
  rc = create_invoices_table();
  if (rc > 0) {
    goto free_and_exit;
  }

free_and_exit:
  return rc;
}

// MARK:- Queries

// Return new invoice id. Set via pointer param.
// return 0 if success, >0 if error.
int db_create_invoice(InvoiceModel *inv) {
  int rc = 1;
  char query[MAX_QUERY_LENGTH];
  sqlite3_stmt *stmt = NULL;

  snprintf(query, sizeof(query),
           "INSERT INTO " INVOICE_TABLE_NAME
           " (sid, user_id, order_sid, product_sid, created_at) VALUES (?, ?, "
           "?, ?, ?)");

  rc = sqlite3_prepare_v2(db, query, -1, &stmt, NULL);
  if (rc != SQLITE_OK) {
    fprintf(stderr, "Error preparing statement: %s\n", sqlite3_errmsg(db));
    goto free_and_exit;
  }

  // Bind values
  sqlite3_bind_text(stmt, 1, inv->sid, -1, SQLITE_STATIC);
  sqlite3_bind_int(stmt, 2, inv->user_id);
  sqlite3_bind_text(stmt, 3, inv->order_sid, -1, SQLITE_STATIC);
  sqlite3_bind_text(stmt, 4, inv->product_sid, -1, SQLITE_STATIC);
  sqlite3_bind_int64(stmt, 5, (sqlite3_int64)inv->created_at);

  rc = sqlite3_step(stmt);
  if (rc != SQLITE_DONE) {
    fprintf(stderr, "Error inserting invoice: %s\n", sqlite3_errmsg(db));
    rc = 2;
    goto free_and_exit;
  }

  // get last insert id
  inv->id = (int)sqlite3_last_insert_rowid(db);
  rc = 0;

free_and_exit:
  if (stmt)
    sqlite3_finalize(stmt);
  return rc;
}

int db_delete_invoice(uint8_t id) {
  int rc = 1;
  char query[MAX_QUERY_LENGTH];
  snprintf(query, sizeof(query),
           "DELETE FROM " INVOICE_TABLE_NAME " WHERE id = %u", id);
  rc = sqlite3_exec(db, query, NULL, NULL, NULL);
  if (rc != SQLITE_OK) {
    fprintf(stderr, "Error deleting invoice: %s\n", sqlite3_errmsg(db));
    goto free_and_exit;
  }
  rc = 0;
free_and_exit:
  return rc;
}

InvoiceModel *db_get_all_invoices(size_t *count) {
  int rc = 1;

  char query[MAX_QUERY_LENGTH];
  snprintf(query, sizeof(query),
           "SELECT * FROM " INVOICE_TABLE_NAME " ORDER BY created_at DESC");

  InvoiceModel *invoices = NULL;

  sqlite3_stmt *stmt = NULL;
  rc = sqlite3_prepare_v2(db, query, -1, &stmt, NULL);
  if (rc != SQLITE_OK) {
    fprintf(stderr, "Error preparing statement: %s\n", sqlite3_errmsg(db));
    goto free_and_exit;
  }

  *count = 0;
  while (sqlite3_step(stmt) == SQLITE_ROW) {
    (*count)++;
  }
  sqlite3_reset(stmt);

  invoices = calloc(*count, sizeof(InvoiceModel));
  if (invoices == NULL) {
    fprintf(stderr, "Error allocating memory for invoices: %s\n",
            sqlite3_errmsg(db));
    goto free_and_exit;
  }

  int i = 0;
  while (sqlite3_step(stmt) == SQLITE_ROW) {
    InvoiceModel inv = {0};
    inv.id = sqlite3_column_int(stmt, 0);
    inv.sid = strdup((char *)sqlite3_column_text(stmt, 1));
    inv.user_id = sqlite3_column_int(stmt, 2);
    inv.order_sid = strdup((char *)sqlite3_column_text(stmt, 3));
    inv.product_sid = strdup((char *)sqlite3_column_text(stmt, 4));
    inv.created_at = (time_t)sqlite3_column_int(stmt, 5);
    invoices[i] = inv;
    i++;
  }

  rc = 0;

free_and_exit:
  sqlite3_finalize(stmt);
  return invoices;
}

#endif