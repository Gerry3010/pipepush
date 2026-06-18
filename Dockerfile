# --- Stage 1: build the web app ---
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# --- Stage 2: build the Go server ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/pipepush-server ./cmd/pipepush-server
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/pipepush ./cmd/pipepush

# --- Stage 3: minimal runtime ---
FROM alpine:3.20
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
