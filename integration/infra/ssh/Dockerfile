FROM linuxserver/openssh-server

# Install dependencies (minimal set for hl download)
RUN apk add --no-cache curl tar gzip

# Install hl (high-performance log viewer)
# Download the musl build for Alpine Linux
ARG HL_VERSION=v0.33.1
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        HL_ARCH="x86_64-musl"; \
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        HL_ARCH="arm64-musl"; \
    else \
        echo "Unsupported architecture: $ARCH" && exit 1; \
    fi && \
    echo "Downloading hl for $HL_ARCH..." && \
    curl -sL "https://github.com/pamburus/hl/releases/download/${HL_VERSION}/hl-linux-${HL_ARCH}.tar.gz" -o /tmp/hl.tar.gz && \
    tar xzf /tmp/hl.tar.gz -C /usr/local/bin && \
    rm /tmp/hl.tar.gz && \
    chmod +x /usr/local/bin/hl && \
    hl --version

# Create directories for logs and scripts
RUN mkdir -p /data /scripts

# Copy log generator script
COPY generate-logs.sh /scripts/generate-logs.sh
RUN chmod 755 /scripts/generate-logs.sh

# Copy authorized keys
COPY authorized_keys /config/authorized_keys
