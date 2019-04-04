FROM golang:1.12 AS builder

COPY . /work
WORKDIR /work
RUN useradd loadwatcher

FROM scratch

LABEL MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"
COPY kubernetes-loadwatcher /usr/sbin/kubernetes-loadwatcher
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/

USER loadwatcher

ENTRYPOINT ["/usr/sbin/kubernetes-loadwatcher", "-logtostderr"]