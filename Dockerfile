FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /auxilia-webserver .

FROM alpine:3.22
RUN adduser -D -H -u 10001 app
USER app
COPY --from=builder /auxilia-webserver /usr/local/bin/auxilia-webserver
EXPOSE 8080
ENTRYPOINT ["auxilia-webserver"]
