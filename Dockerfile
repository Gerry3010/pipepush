# --- Stage 1: build the web app ---
FROM node:22-alpine AS web
WORKDIR /web
# rsvg-convert rasterizes the icon SVG into web/public/ via the npm "prebuild"
# hook (icons are gitignored build artifacts, generated fresh here).
RUN apk add --no-cache rsvg-convert
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# --- Stage 2: build the Go server ---
# Pinned to a patch release: Go links the standard library into the binary,
# so a Go security release only reaches production through a rebuild with it.
# 1.26.11 fixes net/http (GO-2026-6859, SEC-30). The `go` line in go.mod
# enforces the same minimum (the image sets GOTOOLCHAIN=local, so an older
# toolchain fails the build instead of silently producing an unfixed binary).
# Bump both together.
FROM golang:1.26.11-alpine3.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/pipepush-server ./cmd/pipepush-server
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/pipepush ./cmd/pipepush

# --- Stage 3: minimal runtime ---
# Alpine 3.20 is EOL and never got ca-certificates 20261020 (emSign ECC Root
# CA - G3 removed). Keep this on a supported branch.
FROM alpine:3.24
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 pipepush
WORKDIR /app
COPY --from=build /out/pipepush-server /usr/local/bin/pipepush-server
COPY --from=build /out/pipepush /usr/local/bin/pipepush
COPY --from=web /web/dist /app/web
ENV STATIC_DIR=/app/web
ENV PORT=8080
EXPOSE 8080
USER pipepush
ENTRYPOINT ["pipepush-server"]
