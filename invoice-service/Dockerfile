FROM alpine:latest AS builder

RUN apk add --no-cache gcc make wget git libtool automake autoconf unzip musl-dev pkgconf m4

WORKDIR /tmp

RUN wget -O hiredis.zip https://github.com/redis/hiredis/archive/refs/heads/master.zip \
    && unzip hiredis.zip \
    && cd hiredis-master \
    && make && make install

RUN wget -O libuv.tar.gz https://dist.libuv.org/dist/v1.9.1/libuv-v1.9.1.tar.gz \
    && tar -xzf libuv.tar.gz \
    && cd libuv-v1.9.1 \
    && sh autogen.sh \
    && ./configure --prefix=/usr/local \
    && make && make install

WORKDIR /app
COPY . .
RUN git submodule update --init --recursive
RUN gcc -Wall -Wextra -DEPOLL invoice-service/main.c -o service_app $(pkg-config --cflags --libs libuv hiredis)

FROM alpine:latest

COPY --from=builder /usr/local /usr/local
COPY --from=builder /app/service_app /usr/local/bin/service_app

EXPOSE 8080
CMD ["/usr/local/bin/service_app"]