#!/bin/sh
set -e

systemctl daemon-reload
systemctl enable etna-finance.service
systemctl start etna-finance.service