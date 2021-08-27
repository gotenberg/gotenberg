#!/bin/bash

set -e

ARCH=$(dpkg --print-architecture)

if [[ "$ARCH" == "amd64" ]]; then
  curl https://dl.google.com/linux/linux_signing_key.pub | apt-key add -
  echo "deb http://dl.google.com/linux/chrome/deb/ stable main" | tee /etc/apt/sources.list.d/google-chrome.list
  apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends --allow-unauthenticated google-chrome-stable
  mv /usr/bin/google-chrome-stable /usr/bin/chromium
else
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends chromium
fi