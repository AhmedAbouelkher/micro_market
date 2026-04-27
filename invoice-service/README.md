# Invoice Service

## Description

`invoice-service` is a small C microservice that creates, stores, and renders invoices.
It exposes HTTP endpoints for invoice CRUD-lite flows, persists invoice metadata in SQLite,
subscribes to Redis for async invoice generation, and can render invoice PDFs with `PDFGen`.

It runs on top of a tiny C HTTP server and an event loop powered by `libuv`.

## Table of Contents

- [Description](#description)
- [Table of Contents](#table-of-contents)
- [Motivation & Why in C](#motivation--why-in-c)
- [Dependencies Used](#dependencies-used)
- [How to Run Locally](#how-to-run-locally)
- [How to Build & Run with Docker](#how-to-build--run-with-docker)
- [JSON Parsing Note](#json-parsing-note)
- [File Structure](#file-structure)
- [Functions Reference](#functions-reference)

## Motivation & Why in C

This service is written in C for speed, small runtime footprint, and direct control over
memory, files, sockets, and PDF generation.

Why C here:

- Very small binary/runtime compared with heavier web stacks.
- Direct access to SQLite, Redis, and PDF generation libraries.
- Easy to embed inside a container with few moving parts.
- Good fit for a service that mostly does request parsing, persistence, and file output.

Tradeoff: more manual memory and buffer management. This codebase keeps that under control
with small helper functions and explicit cleanup paths.

## Dependencies Used

Runtime and app dependencies:

- `libuv` for event loop integration.
- `hiredis` for Redis async client.
- `sqlite3` for local persistence.
- `httpserver` for the HTTP layer.
- `ulid-c` for invoice SID generation.
- `PDFGen` for PDF creation.

Build-time tools:

- `gcc`
- `make`
- `wget`
- `git`
- `pkgconf` / `pkg-config`
- `unzip`
- `autoconf`, `automake`, `libtool`, `m4`

External sources are installed either by:

- `scripts/install_invoice_service_deps_local.sh` for local setup
- `invoice-service/Dockerfile` for container builds

## How to Run Locally

The recommended local flow is:

1. Install service dependencies.
2. Build from repo root.
3. Run the compiled binary from `invoice-service/`.

### 1) Install dependencies

Use the helper script from repo root:

```bash
bash scripts/install_invoice_service_deps_local.sh
```

What it does:

- initializes required git submodules
- downloads SQLite amalgamation
- installs `hiredis`
- installs `libuv`
- cleans temp archives when finished

### 2) Build the service

From repo root:

```bash
make build_invoice
```

This compiles:

```bash
invoice-service/main.c
invoice-service/libs/ulid/ulid.c
invoice-service/libs/PDFGen/pdfgen.c
```

and produces:

```bash
invoice-service/service_app
```

### 3) Run locally

From repo root:

```bash
make run_invoice
```

Or run the binary directly:

```bash
cd invoice-service
./service_app
```

### Local environment variables

Defaults are used if vars are missing:

- `DB_PATH` default: `invoice.db`
- `HTTP_PORT` default: `8080`
- `REDIS_HOST` default: `localhost`
- `REDIS_PORT` default: `6379`
- `REDIS_CHANNEL` default: `create_invoice`

### Local runtime notes

- Database file is created under `invoice-service/data/`.
- The service opens SQLite, starts HTTP polling, then connects to Redis.
- On shutdown it stops the loop, disconnects Redis, and closes the DB.

## How to Build & Run with Docker

Docker is the recommended way to run this service.

### Build

From repo root:

```bash
make docker-build-invoice
```

This uses `invoice-service/Dockerfile`, which:

- starts from Alpine
- installs compiler and build tools
- installs `hiredis` and `libuv`
- downloads SQLite amalgamation
- clones the needed C libs into `invoice-service/libs/`
- builds `service_app`
- ships only the runtime binary in final image

### Run

Example:

```bash
docker run --rm -p 8080:8080 micro_market-invoice
```

If you want custom env vars:

```bash
docker run --rm -p 8080:8080 \
  -e DB_PATH=invoice.db \
  -e HTTP_PORT=8080 \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e REDIS_CHANNEL=create_invoice \
  micro_market-invoice
```

## JSON Parsing Note

`get_json_value` in `[utils.c](utils.c)` is a tiny hand-rolled JSON extractor.

Why it acts like a mini parser:

- searches for `"key"` inside the payload
- skips whitespace
- checks for `:`
- reads either a quoted string or a simple scalar value
- copies the result into the output buffer

Why this matters:

- avoids pulling in a full JSON dependency
- keeps the service small

Limitations:

- not a real JSON parser
- fragile with nested objects, escaped quotes, arrays, and edge cases
- buffer sizes are fixed

It is enough for the service's current flat request payloads, but it is not general-purpose.

## File Structure

```text
invoice-service/
├── README.md            Service docs
├── Dockerfile           Multi-stage container build
├── main.c               App entrypoint, bootstraps HTTP/Redis/DB
├── api_routes.c         HTTP routing and request handlers
├── db.c                 SQLite models, migrations, queries
├── redis_calls.c        Redis subscription and async invoice jobs
├── gen_pdf.c            PDF generation helpers
├── utils.c              Small shared helpers
├── data/                Generated invoices and local DB files
├── libs/                Vendored or cloned C dependencies
├── .dockerignore        Docker build context cleanup
└── .gitignore           Local ignore rules
```

Purpose of each file:

- `main.c`: starts DB, HTTP server, Redis client, and event loop.
- `api_routes.c`: exposes `/health`, `/invoices`, and `/gen-invoice`.
- `db.c`: owns `InvoiceModel`, migrations, insert/select/delete logic.
- `redis_calls.c`: listens to Redis messages and turns them into invoice jobs.
- `gen_pdf.c`: renders invoice PDF files into `data/`.
- `utils.c`: env access, string helpers, time formatting, JSON extraction.
- `Dockerfile`: builds and packages the service in a container.
- `README.md`: this documentation.

## Functions Reference

| File | Function | Purpose |
| --- | --- | --- |
| `utils.c` | `get_env` | Read env var with fallback default. |
| `utils.c` | `cnstr` | Safe string concatenation helper. |
| `utils.c` | `get_local_time_seconds_since_epoch` | Return local epoch seconds. |
| `utils.c` | `format_time` | Format `time_t` into `YYYY-MM-DD HH:MM:SS`. |
| `utils.c` | `get_json_value` | Extract a flat JSON field by key. |
| `db.c` | `close_db` | Close SQLite handle. |
| `db.c` | `open_db` | Open DB, init ULID, run migrations. |
| `db.c` | `invoice_model_init` | Create invoice model with ULID SID and timestamp. |
| `db.c` | `invoice_model_init_empty` | Create empty invoice model shell. |
| `db.c` | `free_invoice_model` | Free invoice model owned strings. |
| `db.c` | `free_invoice_models` | Free array of invoice models. |
| `db.c` | `json_invoice_model` | Serialize single invoice as JSON. |
| `db.c` | `json_invoice_models` | Serialize invoice array as JSON list. |
| `db.c` | `create_invoices_table` | Create table and unique index. |
| `db.c` | `run_migrations` | Run DB migrations. |
| `db.c` | `db_create_invoice` | Insert invoice into SQLite. |
| `db.c` | `db_delete_invoice` | Delete invoice by numeric ID. |
| `db.c` | `db_get_all_invoices` | Fetch all invoices ordered by newest first. |
| `gen_pdf.c` | `gen_invoice_pdf` | Render and save invoice PDF. |
| `gen_pdf.c` | `draw_text` | Draw text on PDF page with mm-to-point conversion. |
| `api_routes.c` | `poll_http` | Poll HTTP server from libuv timer. |
| `api_routes.c` | `handle_request` | Route requests by path and method. |
| `api_routes.c` | `handle_health_request` | Return service health JSON. |
| `api_routes.c` | `handle_create_invoice_request` | Parse, persist, and return created invoice. |
| `api_routes.c` | `handle_get_invoices_request` | Return all invoices as JSON. |
| `api_routes.c` | `handle_gen_invoice_request` | Generate invoice PDF from request body. |
| `api_routes.c` | `handle_not_found_request` | Return 404 JSON response. |
| `api_routes.c` | `request_target_is` | Compare request path. |
| `api_routes.c` | `request_method_is` | Compare request method. |
| `api_routes.c` | `check_target_and_method` | Match path and method under `/api/v1`. |
| `api_routes.c` | `create_json_error_response` | Build JSON error payload. |
| `api_routes.c` | `send_json_error_response` | Send JSON error response. |
| `api_routes.c` | `parse_invoice_create_request` | Parse invoice creation payload. |
| `redis_calls.c` | `connectCallback` | Subscribe to Redis channel on connect. |
| `redis_calls.c` | `disconnectCallback` | Log Redis disconnect. |
| `redis_calls.c` | `parse_redis_invoice_gen` | Parse Redis invoice job payload. |
| `redis_calls.c` | `invoiceGenCallback` | Handle invoice generation messages. |
| `main.c` | `handle_sigterm` | Graceful shutdown handler. |
| `main.c` | `main` | App entrypoint and bootstrap. |

## API Summary

All HTTP routes live under `/api/v1`.

- `GET /api/v1/health` returns `{"message":"ok"}`
- `POST /api/v1/invoices` creates and stores invoice metadata
- `GET /api/v1/invoices` returns all invoices
- `POST /api/v1/gen-invoice` generates invoice PDF

## Notes

- Request parsing is intentionally simple and assumes flat JSON bodies.
- PDF files are written to `invoice-service/data/`.
- Redis is used for async invoice generation via pub/sub.
