FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy && go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server/main.go

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/server /app/server

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
