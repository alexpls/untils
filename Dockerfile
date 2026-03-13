# build assets
FROM oven/bun:latest AS bun
WORKDIR /app
COPY package.json bun.lock .
RUN bun install
COPY . .
RUN rm -rf public/js/* public/css/*
COPY assets/css ./assets/css
RUN bun run build-css
COPY assets/js ./assets/js
RUN bun run build-js

# build go
FROM golang:1.26-alpine AS go
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/a-h/templ/cmd/templ@latest
COPY . .
COPY --from=bun /app/public ./public
RUN templ generate
RUN go build -o /bin/server ./cmd

# server
FROM scratch
COPY --from=go /bin/server /bin/server
COPY --from=go /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
CMD ["/bin/server", "serve"]
