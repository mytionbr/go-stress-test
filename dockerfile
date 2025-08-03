FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o go-stress-test .

FROM scratch
COPY --from=builder /app/go-stress-test /go-stress-test
ENTRYPOINT ["/go-stress-test"]