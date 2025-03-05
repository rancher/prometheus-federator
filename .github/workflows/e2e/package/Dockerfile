FROM registry.suse.com/bci/golang:1.24 AS helm
RUN zypper -n install git
RUN git -C / clone --branch release-v3.9.0 --depth=1 https://github.com/rancher/helm
RUN make -C /helm

FROM registry.suse.com/bci/golang:1.24

ARG ARCH=amd64
ENV KUBECTL_VERSION v1.21.8

# Install dependencies
RUN zypper -n install awk git docker curl wget nodejs sudo
RUN curl -LO https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl && \
    chmod +x kubectl && mv ./kubectl /usr/local/bin/kubectl
COPY --from=helm ./helm/bin/helm /usr/local/bin/helm

CMD ["echo", "e2e-prometheus-federator"]
