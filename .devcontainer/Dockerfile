FROM golang:1.21-bullseye

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    bash \
    make \
    gcc \
    libc6-dev \
    openssh-client \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Optional: Install kubectl (replace version if needed)
RUN curl -LO https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

# Optional: Install operator-sdk
RUN curl -LO "https://github.com/operator-framework/operator-sdk/releases/download/v1.34.1/operator-sdk_linux_amd64" \
    && chmod +x operator-sdk_linux_amd64 \
    && mv operator-sdk_linux_amd64 /usr/local/bin/operator-sdk

WORKDIR /workspace
