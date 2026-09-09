FROM node:24-alpine AS frontend
ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION}
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY resources ./resources
COPY vite.config.ts ./
RUN npm run build

FROM golang:1.26-alpine AS gobuild
ARG APP_VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/chiririll/savvy-plus/internal/version.Value=${APP_VERSION}" \
    -o /out/savvy ./cmd/savvy

FROM alpine:3.22
ARG APP_VERSION=dev
ENV APP_VERSION=${APP_VERSION} \
    DATA_DIR=/data \
    PUBLIC_DIR=/public \
    LISTEN_ADDR=:80 \
    APP_ENV=production
# Alpine already ships the www-data group (gid 82); only the user is missing.
# hadolint ignore=DL3018
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata libcap wget \
    && adduser -u 82 -S -G www-data -H -D www-data \
    && mkdir -p /data /public \
    && chown www-data:www-data /data
COPY --from=gobuild /out/savvy /usr/local/bin/savvy
COPY --from=frontend /app/public/build /public/build
COPY public/favicon.svg public/robots.txt public/site.webmanifest /public/
RUN setcap 'cap_net_bind_service=+ep' /usr/local/bin/savvy \
    && chown -R www-data:www-data /public
VOLUME /data
EXPOSE 80
USER www-data
ENTRYPOINT ["/usr/local/bin/savvy"]
