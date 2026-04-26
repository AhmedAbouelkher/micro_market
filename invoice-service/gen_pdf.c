#ifndef GEN_PDF_H
#define GEN_PDF_H

#include "db.c"
#include "libs/PDFGen/pdfgen.h"

#define STRING_BUFFER_SIZE 1024
#define FONT_SIZE 12
#define FONT_NAME "Times-Roman"

struct pdf_doc *pdf;

int draw_text(float x, float y, const char *text, int font_size);

int gen_invoice_pdf(InvoiceModel *inv) {
  int rc = 1;

  char title[STRING_BUFFER_SIZE];
  cnstr(title, "Invoice: ", inv->order_sid);
  char created_at[64] = {0};
  char price_per_item[32] = {0};
  char quantity[32] = {0};
  char total[32] = {0};
  format_time(created_at, inv->created_at);
  snprintf(price_per_item, sizeof(price_per_item), "%u", inv->price_per_item);
  snprintf(quantity, sizeof(quantity), "%u", inv->quantity);
  snprintf(total, sizeof(total), "%u", inv->total);

  struct pdf_info info = {0};
  snprintf(info.creator, sizeof(info.creator), "%s", "Micro Market");
  snprintf(info.title, sizeof(info.title), "%s", title);
  snprintf(info.date, sizeof(info.date), "%s", "Today");

  pdf = pdf_create(PDF_A4_WIDTH, PDF_A4_HEIGHT, &info);
  rc = pdf_set_font(pdf, "Times-Roman");
  if (rc != 0)
    goto clean_and_exit;

  pdf_append_page(pdf);

  draw_text(20, 20, title, 24);
  draw_text(20, 35, "Created:", 12);
  draw_text(50, 35, created_at, 12);
  draw_text(20, 50, "Invoice SID:", 12);
  draw_text(60, 50, inv->sid, 12);

  draw_text(20, 70, "Customer", 14);
  draw_text(20, 85, "Name:", 12);
  draw_text(45, 85, inv->user_name, 12);
  draw_text(20, 100, "Email:", 12);
  draw_text(45, 100, inv->user_email, 12);

  draw_text(20, 120, "Item", 14);
  draw_text(20, 135, "Product:", 12);
  draw_text(55, 135, inv->product_name, 12);
  draw_text(20, 150, "Product SID:", 12);
  draw_text(65, 150, inv->product_sid, 12);
  draw_text(20, 165, "Price:", 12);
  draw_text(45, 165, price_per_item, 12);
  draw_text(20, 180, "Qty:", 12);
  draw_text(40, 180, quantity, 12);
  draw_text(20, 205, "Total:", 14);
  draw_text(45, 205, total, 14);

  // Saving the pdf to a file
  // check if there is any error in the pdf
  if (pdf_get_err(pdf, &rc) != NULL) {
    fprintf(stderr, "ERROR MESSAGE: (%d) %s\n", rc, pdf_get_err(pdf, &rc));
    goto clean_and_exit;
  }

  char filename[STRING_BUFFER_SIZE];
  snprintf(filename, sizeof(filename), "%s/%s.pdf", "data", inv->order_sid);

  rc = pdf_save(pdf, filename);
  if (rc != 0)
    goto clean_and_exit;

  printf("Invoice PDF saved to: %s\n", filename);

  rc = 0;

clean_and_exit:
  pdf_clear_err(pdf);
  pdf_destroy(pdf);
  return rc;
}

int draw_text(float x, float y, const char *text, int font_size) {
  float xx = PDF_MM_TO_POINT(x);
  float yy = PDF_A4_HEIGHT - PDF_MM_TO_POINT(y);
  return pdf_add_text(pdf, NULL, text, font_size, xx, yy, PDF_BLACK);
}

#endif