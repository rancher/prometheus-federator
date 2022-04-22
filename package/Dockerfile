FROM registry.suse.com/bci/bci-micro:15.3
RUN echo 'prometheus:x:1000:1000::/home/prometheus:/bin/bash' >> /etc/passwd && \
    echo 'prometheus:x:1000:' >> /etc/group && \
    mkdir /home/prometheus && \
    chown -R prometheus:prometheus /home/prometheus
COPY bin/prometheus-federator /usr/bin/
USER prometheus
CMD ["prometheus-federator"]
