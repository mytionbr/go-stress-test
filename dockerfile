FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o go-stress-test .

FROM alpine:latest
COPY --from=builder /app/go-stress-test /go-stress-test
RUN apk add --no-cache ca-certificates
ENTRYPOINT ["/go-stress-test"]