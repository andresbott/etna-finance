#!/bin/sh
set -e

USER=etna-finance
GROUP=etna-finance
DATA_DIR=/usr/share/etna-finance/data

# Create group if it doesn't exist
if ! getent group "$GROUP" >/dev/null; then
  groupadd --system "$GROUP"
fi

# Create user if it doesn't exist
if ! getent passwd "$USER" >/dev/null; then
  useradd \
    --system \
    --no-create-home \
    --shell /usr/sbin/nologin \
    --gid "$GROUP" \
    "$USER"
fi

# Create data directory if it doesn't exist
if [ ! -d "$DATA_DIR" ]; then
  mkdir -p "$DATA_DIR"
  chown "$USER:$GROUP" "$DATA_DIR"
  chmod 750 "$DATA_DIR"
fi