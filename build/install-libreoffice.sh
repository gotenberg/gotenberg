#!/bin/bash

set -e

# By defining an explicit version we get a build time error if LibreOffice's version has changed.
# This may be useful in triggering some regression testing before updating to a new LibreOffice version.
LIBREOFFICE_VERSION="$1"

# This could be useful when we want to know what versions are available after package updates.
apt-cache policy libreoffice
UBUNTU_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends -t focal-backports libreoffice=${LIBREOFFICE_VERSION}
