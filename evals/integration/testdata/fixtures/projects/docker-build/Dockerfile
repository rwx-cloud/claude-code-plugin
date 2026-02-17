FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o server .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
