FROM golang:1.24-bullseye

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
    gnupg \
    lsb-release \
    iproute2 \
    iptables \
    ca-certificates \
    software-properties-common \
    && rm -rf /var/lib/apt/lists/*

# Install kubectl
RUN curl -LO https://dl.k8s.io/release/v1.32.0/bin/linux/amd64/kubectl \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

# Install operator-sdk
RUN curl -LO "https://github.com/operator-framework/operator-sdk/releases/download/v1.39.2/operator-sdk_linux_amd64" \
    && chmod +x operator-sdk_linux_amd64 \
    && mv operator-sdk_linux_amd64 /usr/local/bin/operator-sdk

# Install Kind
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64 \
    && chmod +x ./kind \
    && mv ./kind /usr/local/bin/kind

# Install Docker CLI (for building images)
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian bullseye stable" > /etc/apt/sources.list.d/docker.list \
    && apt-get update \
    && apt-get install -y docker-ce-cli \
    && rm -rf /var/lib/apt/lists/*

# Install Kustomize
RUN curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash \
    && mv kustomize /usr/local/bin/

# Install Helm
RUN curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Add Docker group for container access (Optional but useful in some setups)
RUN groupadd docker || true

# Add bash aliases and completions (optional)
RUN echo 'alias k=kubectl' >> ~/.bashrc \
    && echo 'source <(kubectl completion bash)' >> ~/.bashrc \
    && echo 'complete -F __start_kubectl k' >> ~/.bashrc

WORKDIR /workspace