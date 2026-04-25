#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "hiredis/adapters/libuv.h"
#include "hiredis/async.h"
#include "hiredis/hiredis.h"

#define HTTPSERVER_IMPL
#include "libs/httpserver.h/httpserver.h"

#include "libs/PDFGen/pdfgen.h" // TODO: use PDFGen to generate the invoice

// https://stackoverflow.com/questions/39486327/stdout-being-buffered-in-docker-container

#define REDIS_CHANNEL "service_app"

struct http_server_s *server;
redisAsyncContext *redisCtx;
uv_loop_t *loop;

void subCallback(redisAsyncContext *c, void *r, void *privdata);

// MARK: - Utils

void handle_sigterm(int signum) {
  (void)signum;
  free(server);
  if (loop) {
    uv_stop(loop);
  }
  redisAsyncDisconnect(redisCtx);
}

const char *get_env(const char *env_name, const char *default_value) {
  char *env_value = getenv(env_name);
  if (!env_value) {
    return default_value;
  }
  return env_value;
}

// MARK: - Redis

// TODO: handle the received messages from Redis and process the request.
void subCallback(redisAsyncContext *c, void *r, void *privdata) {
  redisReply *reply = r;
  if (reply == NULL)
    return;
  printf("SUB RESPONSE: %s\n", reply->str);
  if (reply->type == REDIS_REPLY_ARRAY) {
    for (int j = 0; j < reply->elements; j++) {
      printf("%u) %s\n", j, reply->element[j]->str);
    }
  }
}

void connectCallback(const redisAsyncContext *c, int status) {
  if (status != REDIS_OK) {
    fprintf(stderr, "failed to connect to Redis: %s\n", c->errstr);
    return;
  }
  fprintf(stdout, "connected to Redis\n");
  if (redisAsyncCommand(redisCtx, subCallback, NULL, "SUBSCRIBE %s",
                        REDIS_CHANNEL) != REDIS_OK) {
    fprintf(stderr, "failed to subscribe to Redis channel: %s\n",
            redisCtx->errstr);
    redisAsyncFree(redisCtx);
  }
}

void disconnectCallback(const redisAsyncContext *c, int status) {
  if (status != REDIS_OK) {
    fprintf(stderr, "failed to disconnect from Redis: %s\n", c->errstr);
    return;
  }

  printf("Disconnected from Redis\n");
}

// MARK: - API Routes

void poll_http(uv_timer_t *handle) {
  http_server_t *server = handle->data;
  while (http_server_poll(server) > 0)
    ;
}

// TODO: check the request uri and method.
void handle_request(struct http_request_s *request) {
  http_string_t route = http_request_target(request);
  printf("Request: %s\n", route.buf);

  struct http_response_s *response = http_response_init();
  http_response_status(response, 200);
  http_response_header(response, "Content-Type", "application/json");
  char *body = "{\"message\": \"Pong\"}";
  http_response_body(response, body, strlen(body));
  http_respond(request, response);
}

int main(void) {
  signal(SIGTERM, handle_sigterm);
#ifdef __linux__
  signal(SIGINT, handle_sigterm);
#endif

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
  uv_timer_start(&http_timer, poll_http, 0, 100);
  printf("Listening on port %d\n", port_int);

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
