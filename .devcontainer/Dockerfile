FROM mcr.microsoft.com/devcontainers/base:debian

ARG GO_VERSION=1.24.4
ARG NODE_VERSION=20

# Install Go
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:/home/vscode/go/bin:${PATH}"
ENV GOPATH="/home/vscode/go"

# Install Node.js
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && apt-get install -y nodejs

# Install Claude Code
RUN npm install -g @anthropic-ai/claude-code

# Install iptables for firewall
RUN apt-get update && apt-get install -y iptables make bubblewrap socat \
    && rm -rf /var/lib/apt/lists/*
