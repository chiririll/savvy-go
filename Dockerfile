ARG APP_VERSION=dev
ARG APP_ENV=production

# Build the frontend
FROM node:24-alpine AS frontend
ARG APP_VERSION
ARG APP_ENV
ENV APP_VERSION=${APP_VERSION}
ENV APP_ENV=${APP_ENV}
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY resources ./resources
COPY vite.config.ts ./
RUN npm run build

# Build the backend
FROM golang:1.26-alpine AS gobuild
ARG APP_VERSION
ARG APP_ENV
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X savvy-go/internal/version.Value=${APP_VERSION} -X savvy-go/internal/version.Env=${APP_ENV}" \
    -o /out/savvy-go ./cmd/savvy-go

# Build the final image
FROM alpine:3.22
ENV DATA_DIR=/data \
    PUBLIC_DIR=/public \
    LISTEN_ADDR=:80
# wget (used by the healthcheck) is provided by busybox
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata libcap \
    && adduser -u 82 -S -G www-data -H -D www-data \
    && mkdir /data \
    && chown www-data:www-data /data
COPY --from=gobuild /out/savvy-go /usr/local/bin/savvy-go
COPY --chown=www-data:www-data public /public
COPY --chown=www-data:www-data --from=frontend /app/public/build /public/build
RUN setcap 'cap_net_bind_service=+ep' /usr/local/bin/savvy-go
VOLUME /data
EXPOSE 80
USER www-data
ENTRYPOINT ["/usr/local/bin/savvy-go"]
