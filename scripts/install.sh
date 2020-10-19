#!/usr/bin/env bash

set -e

APP_DIR=$(cd "$(dirname "$0")" || exit; pwd)

sed "s/%app_dir%/${APP_DIR//'/'/'\/'}/g" omil.service.template > omil.service
cp ./omil.service /usr/lib/systemd/system
cp ./omil.service /lib/systemd/system
systemctl enable omil.service
systemctl start omil.service
