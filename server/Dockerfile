FROM golang:1.7.5-alpine

ADD ./vendor /go/src/
ADD ./*.go /app/

WORKDIR /app/
RUN go build -o /app/main

EXPOSE 5001
EXPOSE 5002
EXPOSE 80

CMD ["/app/main"]
