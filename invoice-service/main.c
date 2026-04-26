#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "hiredis/adapters/libuv.h"
#include "hiredis/async.h"
#include "libs/sqlite3/sqlite3.c"

#define HTTPSERVER_IMPL
#include "libs/httpserver/httpserver.h"

#include "api_routes.c"
#include "db.c"
#include "redis_calls.c"
#include "utils.c"

// https://stackoverflow.com/questions/39486327/stdout-being-buffered-in-docker-container

struct http_server_s *server;
redisAsyncContext *redisCtx;
uv_loop_t *loop;

void handle_sigterm(int signum) {
  (void)signum;
  free(server);
  if (loop) {
    uv_stop(loop);
  }
  redisAsyncDisconnect(redisCtx);
  close_db();
}

int main(void) {
  signal(SIGTERM, handle_sigterm);
#ifdef __linux__
  signal(SIGINT, handle_sigterm);
#endif

  const char *db_path = get_env("DB_PATH", "invoice.db");
  int rc = open_db(db_path);
  if (rc != 0) {
    fprintf(stderr, "Error opening database: %s\n", sqlite3_errmsg(db));
    return 1;
  }
  const char *http_port = get_env("HTTP_PORT", "8080");
  int port_int = atoi(http_port);
  if (port_int <= 0 || port_int > 65535) {
    fprintf(stderr,
            "HTTP_PORT environment variable is not a valid port number\n");
    return 1;
  }
  const char *redis_host = get_env("REDIS_HOST", "localhost");
  const char *redis_port = get_env("REDIS_PORT", "6379");
  int redis_port_int = atoi(redis_port);
  if (redis_port_int <= 0 || redis_port_int > 65535) {
    fprintf(stderr,
            "REDIS_PORT environment variable is not a valid port number\n");
    return 1;
  }

  // create database directory if it does not exist
  struct stat st = {0};
  if (stat("data", &st) == -1) {
    if (mkdir("data", 0755) != 0) {
      fprintf(stderr, "Error creating database directory: data\n");
      return 1;
    }
  }

  // >>>>>>>>>>> DB <<<<<<<<<<
  char full_db_path[1024];
  cnstr(full_db_path, "data/", db_path);
  if (open_db(full_db_path) > 0) {
    return 1;
  }
  printf("DB was opened successfully at %s\n", db_path);

  // >>>>>>>>>>> HTTP Server <<<<<<<<<<
  server = http_server_init(port_int, handle_request);
  http_server_listen_poll(server);

  loop = uv_default_loop();
  if (!loop) {
    fprintf(stderr, "Error: Cannot get current run loop\n");
    return 1;
  }

  uv_timer_t http_timer;
  uv_timer_init(loop, &http_timer);
  http_timer.data = server;
  uv_timer_start(&http_timer, poll_http, 0, 5);
  printf("HTTP Server started http://localhost:%d\n", port_int);

  // >>>>>>>>>>> Redis Client <<<<<<<<<<

  redisCtx = redisAsyncConnect(redis_host, redis_port_int);
  if (redisCtx == NULL || redisCtx->err) {
    fprintf(stderr, "failed to connect to Redis: %s\n", redisCtx->errstr);
    return 1;
  }

  redisLibuvAttach(redisCtx, loop);

  redisAsyncSetConnectCallback(redisCtx, connectCallback);
  redisAsyncSetDisconnectCallback(redisCtx, disconnectCallback);

  uv_run(loop, UV_RUN_DEFAULT);

  return 0;
}
