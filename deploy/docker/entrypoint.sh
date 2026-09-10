#!/bin/sh
set -eu
# Single Go process: migrations and the HTTP server start in cmd/savvy.
exec /usr/local/bin/savvy-go
