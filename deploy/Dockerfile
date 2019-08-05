FROM golang:1.12.6 AS build-env

WORKDIR /src/github.com/AliyunContainerService/alibaba-cloud-metrics-adapter
ENV GOPATH /:/src/github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/vendor

ADD . /src/github.com/AliyunContainerService/alibaba-cloud-metrics-adapter
RUN apt-get update -y && apt-get install gcc ca-certificates
RUN make


FROM alpine:3.10

COPY --from=build-env /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
COPY --from=build-env /src/github.com/AliyunContainerService/alibaba-cloud-metrics-adapter/alibaba-cloud-metrics-adapter /
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN chmod +x /alibaba-cloud-metrics-adapter

ENTRYPOINT ["/alibaba-cloud-metrics-adapter"]