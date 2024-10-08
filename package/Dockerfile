# Image that provides cross compilation tooling.
FROM --platform=$BUILDPLATFORM rancher/mirrored-tonistiigi-xx:1.3.0 AS xx

FROM --platform=$BUILDPLATFORM registry.suse.com/bci/golang:1.22 AS helm

# Clone repository once, and reuse it for target archs.
ARG HELM_VERSION=release-v3.9.0
ADD --keep-git-dir=true https://github.com/rancher/helm.git#${HELM_VERSION} /helm
RUN cd /helm && go mod download

COPY --from=xx / /

# Cross-compile instead of emulating the compilation on the target arch.
ARG TARGETPLATFORM
RUN xx-go --wrap && mkdir -p /run/lock
RUN make -C /helm

RUN xx-verify --static /helm/bin/helm

FROM registry.suse.com/bci/golang:1.22 AS builder

# Allow chart version config
ARG TARGETPLATFORM
ARG EMBEDED_CHART_VERSION=0.3.4
ARG TAG=''
ARG REPO=''
ENV EMBEDED_CHART_VERSION=$EMBEDED_CHART_VERSION TAG=$TAG REPO=$REPO

WORKDIR /usr/src/app
COPY --from=helm ./helm/bin/helm /usr/local/bin/
RUN zypper -n install git vim less file curl wget patch
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ./scripts/build

FROM registry.suse.com/bci/bci-micro:latest
RUN echo 'prometheus:x:1000:1000::/home/prometheus:/bin/bash' >> /etc/passwd && \
    echo 'prometheus:x:1000:' >> /etc/group && \
    mkdir /home/prometheus && \
    chown -R prometheus:prometheus /home/prometheus
COPY --from=helm ./helm/bin/helm /usr/local/bin/
COPY --from=builder /usr/src/app/build/bin/prometheus-federator /usr/bin/
USER prometheus
CMD ["prometheus-federator"]
