ARG APP_VERSION=v0.0.0
ARG APP_ENV=production

# Build the frontend
FROM node:24-alpine AS frontend
ENV APP_VERSION=${APP_VERSION}
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY resources ./resources
COPY vite.config.ts ./
RUN npm run build

# Build the backend
FROM golang:1.26-alpine AS gobuild
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X savvy-go/internal/version.Value=${APP_VERSION} -X savvy-go/internal/version.Env=${APP_ENV}" \
    -o /out/savvy-go ./cmd/savvy-go

# Build the final image
FROM alpine:3.22
ENV DATA_DIR=/data \
    PUBLIC_DIR=/public \
    LISTEN_ADDR=:80
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata libcap wget \
    && adduser -u 82 -S -G www-data -H -D www-data \
    && mkdir -p /data /public \
    && chown www-data:www-data /data
COPY --from=gobuild /out/savvy-go /usr/local/bin/savvy-go
COPY --from=frontend /app/public/build /public/build
COPY public/favicon.svg public/robots.txt public/site.webmanifest /public/
RUN setcap 'cap_net_bind_service=+ep' /usr/local/bin/savvy-go \
    && chown -R www-data:www-data /public
VOLUME /data
EXPOSE 80
USER www-data
ENTRYPOINT ["/usr/local/bin/savvy-go"]
