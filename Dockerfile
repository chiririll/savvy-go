ARG APP_VERSION=dev
ARG APP_ENV=production

# Build the frontend
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend
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
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS gobuild
ARG TARGETOS
ARG TARGETARCH
ARG APP_VERSION
ARG APP_ENV
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd
COPY internal ./internal
# The frontend is embedded into the binary (build tag "embed").
COPY public ./internal/webui/dist
COPY --from=frontend /app/public/build ./internal/webui/dist/build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -tags embed -trimpath \
    -ldflags="-s -w -X savvy-go/internal/config.DefaultListen=:80 -X savvy-go/internal/version.Value=${APP_VERSION} -X savvy-go/internal/version.Env=${APP_ENV}" \
    -o /out/savvy-go ./cmd/savvy-go

# Build the final image
FROM alpine:3.22
# wget (used by the healthcheck) is provided by busybox
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata libcap \
    && adduser -u 82 -S -G www-data -H -D www-data \
    && mkdir /data \
    && chown www-data:www-data /data
COPY --from=gobuild /out/savvy-go /usr/local/bin/savvy-go
RUN setcap 'cap_net_bind_service=+ep' /usr/local/bin/savvy-go
VOLUME /data
EXPOSE 80
USER www-data
ENTRYPOINT ["/usr/local/bin/savvy-go"]
