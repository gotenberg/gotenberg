#!/bin/sh

# This script installs custom fonts if configured to do so and then deploys Gotenberg

FONT_DIR="/home/gotenberg/custom_fonts"
FLAG_DIR="${FONT_DIR}/flags"
TEMP_DIR="/home/gotenberg/temp"
TEMP_FILE="${TEMP_DIR}/Fonts.tar.gz"

# Function to install fonts
install_fonts() {
    if [ -z "${FONT_URL}" ]; then
        echo "FONT_URL is not set. Skipping font installation."
        return
    fi
    # Create directories if not already created
    mkdir -p ${FONT_DIR}
    mkdir -p ${FLAG_DIR}
    mkdir -p ${TEMP_DIR}
    # Download the custom fonts file. Note - Must be .tar.gz
    curl --retry 3 --retry-delay 5 -o ${TEMP_FILE} "${FONT_URL}" || { echo "Curl command failed after 3 attempts. Ensure that file exists and can be downloaded from ${FONT_URL}."; return; }  && \
    # Extract the font files to the appropriate directory.
    tar -zxvf ${TEMP_FILE} -C ${FONT_DIR} > /dev/null || { echo 'Cannot extract fonts file. Make sure that file is .tar.gz'; return; } && \
    # Refresh the font cache so that the new fonts are recognised.
    fc-cache -fv > /dev/null && \
    # Remove the Fonts.tar.gz file.
    rm ${TEMP_FILE}
    if [ -n "${FONT_FILE_VERSION}" ]; then
        # Create a flag file to indicate that the fonts for this version were installed
        touch "${FLAG_DIR}/${FONT_FILE_VERSION}.flag"
        echo "Custom Fonts version ${FONT_FILE_VERSION} successfully installed."
    else
        echo "Custom Fonts successfully installed."
    fi
}

# Logic to handle installation based on different variables
if [ "$INSTALL_FONTS_ON_DEPLOY" = "true" ]; then
    install_fonts
elif [ -n "${FONT_FILE_VERSION}" ]; then
    # Define the flag file path. The flag file is used to check whether the current FONT_FILE_VERSION has already been installed.
    FLAG_FILE="${FLAG_DIR}/${FONT_FILE_VERSION}.flag"
    # Proceed only if the flag file does not exist which indicates that this FONT_FILE_VERSION has not yet been installed.
    if [ ! -f "${FLAG_FILE}" ]; then
        install_fonts
    else
        echo "${FONT_FILE_VERSION} already installed."
    fi
else
    echo "Skipping font installation. Neither INSTALL_FONTS_ON_DEPLOY nor FONT_FILE_VERSION is set."
fi

# Start Tini and pass Gotenberg to it
exec /usr/bin/tini -- gotenberg
