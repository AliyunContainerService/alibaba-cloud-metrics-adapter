FROM alpine:3.8

RUN apk update \
    && apk add ca-certificates tzdata \
    && rm -rf /var/cache/apk/* \
    && update-ca-certificates

ADD alibaba-cloud-metrics-adapter /alibaba-cloud-metrics-adapter

RUN chmod +x /alibaba-cloud-metrics-adapter

ENTRYPOINT ["/alibaba-cloud-metrics-adapter"]