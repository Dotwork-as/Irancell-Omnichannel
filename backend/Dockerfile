# Irancell Omnichannel — Backend (Go + Fiber + SQLite, pure Go driver)
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.io,direct
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.21
# Non-root runtime (hardening): the named volume inherits this
# ownership on first use; bind-mount users must chown the host dir.
RUN addgroup -S app && adduser -S app -G app \
    && mkdir -p /data && chown -R app:app /data
COPY --from=build --chown=app:app /out/server /app/server
USER app

# Runtime configuration (overridable via compose environment).
ENV PORT=8080 \
    DATABASE_PATH=/data/omnichannel.db
VOLUME ["/data"]
EXPOSE 8080

# Migrations and seed data are applied automatically at startup
# (cmd/server/main.go). No secrets are baked into this image.
ENTRYPOINT ["/app/server"]
