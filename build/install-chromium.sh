#!/bin/bash

# Installs Chromium from Debian sources as ongoing maintenance in Ubuntu has been give to 'snap'.
# There are issues running snap in a Dockerized environment, which makes the following necessary.

set -e

# By defining an explicit version we get a build time error if Chromium's version has changed.
# This may be useful in triggering some regression testing before updating to a new Chromium version.
CHROMIUM_VERSION="$1"

apt-get install debian-archive-keyring

# Add Debian bullseye package sources so they are available to the package manager.
cat <<EOF | tee /etc/apt/sources.list.d/debian-stable.list
deb [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] http://deb.debian.org/debian bullseye main
deb-src [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] http://deb.debian.org/debian bullseye main

deb [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] http://deb.debian.org/debian bullseye-updates main
deb-src [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] http://deb.debian.org/debian bullseye-updates main

deb [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] http://deb.debian.org/debian-security/ bullseye-security main
deb-src [signed-by=/usr/share/keyrings/debian-archive-keyring.gpg] http://deb.debian.org/debian-security/ bullseye-security main
EOF

# Override Ubuntu package manager source preferences for Chromium-related packages
cat <<EOF | tee /etc/apt/preferences.d/chromium.pref

# Packages included:
#   all packages beginning with 'chromium', e.g.'chromium', 'chromium-browser', etc.
#     OR
#   exact match 'libwebpmux3' (required as chromium depends on non-Ubuntu variant)
Package: chromium* libwebpmux3
# From man:
#   500 <= P < 990
#     causes a version to be installed unless there is a version available belonging to the
#     target release or the installed version is more recent
Pin: release o=Debian
Pin-Priority: 750

# This seems to be useful for Chromium dependencies.
Package: *
Pin: release o=Debian
# From man:
#   0 < P < 100
#     causes a version to be installed only if there is no installed version of the package
Pin-Priority: 50
EOF

apt-get update -qq
# This could be useful when we want to know what versions are available after package updates.
apt-cache policy chromium
UBUNTU_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends libatomic1 libwebpmux3 chromium=${CHROMIUM_VERSION}
