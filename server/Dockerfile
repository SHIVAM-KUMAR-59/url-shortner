# --- Build stage ---
FROM golang:1.26-alpine AS build

WORKDIR /app

# Cache dependencies separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the server binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# Build goose (migration tool) so it's available at runtime without
# needing network access inside the final image.
RUN go install github.com/pressly/goose/v3/cmd/goose@latest && \
    cp "$(go env GOPATH)/bin/goose" /out/goose

# --- Final stage ---
FROM alpine:3.20

# Postgres client libs aren't needed since goose/pgx are statically built,
# but ca-certificates is needed for any outbound TLS calls the app makes.
RUN apk add --no-cache ca-certificates bash

WORKDIR /app

COPY --from=build /out/server ./server
COPY --from=build /out/goose ./goose
COPY migrations ./migrations
COPY scripts/entrypoint.sh ./entrypoint.sh

RUN chmod +x ./entrypoint.sh ./goose ./server

EXPOSE 8080

ENTRYPOINT ["./entrypoint.sh"]
