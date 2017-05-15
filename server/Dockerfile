FROM golang:1.7.5-alpine

ADD ./vendor /go/src/
ADD ./*.go /app/

WORKDIR /app/
RUN go build -o /app/main
RUN mkdir -p /var/www/.well-known/
RUN echo "This is the index for .well-known" >> /var/www/.well-known/index.html

EXPOSE 80

CMD ["/app/main"]
