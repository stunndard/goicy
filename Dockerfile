FROM golang:1.13-alpine

WORKDIR /app


COPY . /app

RUN go mod download

RUN go build ./goicy.go


ENTRYPOINT [ "/app/goicy" ]
