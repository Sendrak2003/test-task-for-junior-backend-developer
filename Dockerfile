FROM golang:1.23.0-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

# Install swag CLI for OpenAPI generation
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4

COPY . .

# Generate OpenAPI spec from annotations
RUN swag init \
    --generalInfo cmd/api/main.go \
    --output internal/transport/http/docs/generated \
    --outputTypes json \
    --parseDependency \
    --parseInternal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/taskservice ./cmd/api

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/taskservice /app/taskservice

EXPOSE 8080

CMD ["/app/taskservice"]
