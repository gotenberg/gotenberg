#!/bin/bash

# This entrypoint allows us to set the UID and GID of the host user so that
# our testing environment does not override files permissions from the host.
# Credits: https://github.com/thecodingmachine/docker-images-php.

set +e
mkdir testing_file_system_rights.foo
chmod 700 testing_file_system_rights.foo
su gotenberg -c "touch testing_file_system_rights.foo/foo > /dev/null 2>&1"
HAS_CONSISTENT_RIGHTS=$?

if [[ "$HAS_CONSISTENT_RIGHTS" != "0" ]]; then
    # If not specified, the DOCKER_USER is the owner of the current working directory (heuristic!).
    DOCKER_USER=`ls -dl $(pwd) | cut -d " " -f 3`
else
    # macOs or Windows.
    # Note: in most cases, we don't care about the rights (they are not respected).
    FILE_OWNER=`ls -dl testing_file_system_rights.foo/foo | cut -d " " -f 3`
    if [[ "$FILE_OWNER" == "root" ]]; then
        # If root, we are likely on a Windows host.
        # All files will belong to root, but it does not matter as everybody can write/delete
        # those (0777 access rights).
        DOCKER_USER=gotenberg
    else
        # In case of a NFS mount (common on macOS), the created files will belong to the NFS user.
        DOCKER_USER=$FILE_OWNER
    fi
fi

rm -rf testing_file_system_rights.foo
set -e
unset HAS_CONSISTENT_RIGHTS

# Note: DOCKER_USER is either a username (if the user exists in the container),
# otherwise a user ID (a user from the host).

# DOCKER_USER is an ID.
if [[ "$DOCKER_USER" =~ ^[0-9]+$ ]] ; then
    # Let's change the gotenberg user's ID in order to match this free ID.
    usermod -u $DOCKER_USER -G sudo gotenberg
    DOCKER_USER=gotenberg
fi

DOCKER_USER_ID=`id -ur $DOCKER_USER`

# Fix access rights to stdout and stderr.
set +e
chown $DOCKER_USER /proc/self/fd/{1,2}
set -e

# Install modules.
set -x
go mod download
go mod tidy
set +x

# Run the command with the correct user.
exec "sudo" "-E" "-H" "-u" "#$DOCKER_USER_ID" "$@"