FROM registry.suse.com/bci/golang:1.22 AS helm
RUN zypper -n install git
RUN git -C / clone --branch release-v3.9.0 --depth=1 https://github.com/rancher/helm
RUN make -C /helm

FROM registry.suse.com/bci/golang:1.22 as builder
WORKDIR /usr/src/app
COPY --from=helm ./helm/bin/helm /usr/local/bin/
RUN zypper -n install git vim less file curl wget patch
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

FROM registry.suse.com/bci/bci-micro:15.5
RUN echo 'prometheus:x:1000:1000::/home/prometheus:/bin/bash' >> /etc/passwd && \
    echo 'prometheus:x:1000:' >> /etc/group && \
    mkdir /home/prometheus && \
    chown -R prometheus:prometheus /home/prometheus
COPY --from=helm ./helm/bin/helm /usr/local/bin/
COPY --from=builder /usr/src/app/bin/prometheus-federator /usr/bin/
USER prometheus
CMD ["prometheus-federator"]
