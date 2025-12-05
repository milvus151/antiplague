FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o file-storing ./main.go

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/file-storing .

EXPOSE 8082

CMD ["./file-storing"]
