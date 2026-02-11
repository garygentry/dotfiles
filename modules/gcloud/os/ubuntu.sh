#!/usr/bin/env bash
# gcloud/os/ubuntu.sh - Set up Google Cloud apt repository on Ubuntu

if apt-cache show google-cloud-cli &>/dev/null 2>&1; then
    log_info "Google Cloud apt repo already configured"
    return 0
fi

log_info "Adding Google Cloud apt repository..."

# Add Google Cloud keyring
curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg \
    | sudo gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg 2>/dev/null

# Add apt source
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
    | sudo tee /etc/apt/sources.list.d/google-cloud-sdk.list > /dev/null

sudo apt-get update -qq
