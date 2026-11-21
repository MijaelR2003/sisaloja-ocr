FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o sisaloja-ocr .

FROM alpine:3.20
RUN apk add --no-cache tesseract-ocr tesseract-ocr-data-spa ca-certificates
WORKDIR /app
COPY --from=builder /app/sisaloja-ocr /app/sisaloja-ocr
EXPOSE 8090
ENV PORT=8090
CMD ["/app/sisaloja-ocr"]
