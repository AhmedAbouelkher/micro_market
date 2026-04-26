FROM alpine:latest AS builder

RUN apk add --no-cache gcc make wget git libtool automake autoconf unzip musl-dev pkgconf m4

WORKDIR /tmp

# Install hiredis
RUN wget -O hiredis.zip https://github.com/redis/hiredis/archive/refs/heads/master.zip \
    && unzip hiredis.zip \
    && cd hiredis-master \
    && make && make install

# Install libuv
RUN wget -O libuv.tar.gz https://dist.libuv.org/dist/v1.9.1/libuv-v1.9.1.tar.gz \
    && tar -xzf libuv.tar.gz \
    && cd libuv-v1.9.1 \
    && sh autogen.sh \
    && ./configure --prefix=/usr/local \
    && make && make install

WORKDIR /app

COPY . .

RUN mkdir -p libs

# Install SQLite3
RUN wget -O sqlite_amalgamation.zip https://sqlite.org/2026/sqlite-amalgamation-3530000.zip \
    && unzip sqlite_amalgamation.zip -d sqlite_amalgamation \
    && mv sqlite_amalgamation/* libs/sqlite3

# Clone dependencies
RUN git clone https://github.com/skeeto/ulid-c.git libs/ulid \
    && git clone https://github.com/AndreRenaud/PDFGen.git libs/PDFGen \
    && git clone https://github.com/AhmedAbouelkher/httpserver.git libs/httpserver

RUN gcc -Wall -Wextra -DEPOLL main.c libs/ulid/ulid.c libs/PDFGen/pdfgen.c \
    -o service_app $(pkg-config --cflags --libs libuv hiredis)

FROM alpine:latest

COPY --from=builder /usr/local /usr/local
COPY --from=builder /app/service_app /usr/local/bin/service_app

EXPOSE 8080
CMD ["/usr/local/bin/service_app"]