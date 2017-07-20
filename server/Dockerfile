FROM golang:1.7.5-alpine AS builder

WORKDIR /app/

ADD ./vendor /go/src/
ADD ./*.go /app/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main

# MULTI-STAGE BUILD PATTERN

FROM alpine:latest

RUN set -x \
      && mkdir -p /var/www/.well-known/ \
      && mkdir -p /etc/auto-kubernetes-lets-encrypt/certs/ \
      && echo "This is the index for .well-known" >> /var/www/.well-known/index.html

WORKDIR /app/
COPY --from=builder /app/main /app/main

EXPOSE 80
CMD ["/app/main"]
