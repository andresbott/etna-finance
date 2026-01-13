#!/bin/sh
set -e

systemctl stop etna-finance.service || true
systemctl disable etna-finance.service || true
systemctl daemon-reload