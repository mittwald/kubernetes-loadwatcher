FROM golang:1.12 AS builder

COPY . /work
WORKDIR /work
RUN useradd loadwatcher
RUN go mod download && \
    cd cmd && \
    CGO_ENABLED=0 go build -ldflags="-w -s" -o /work/loadwatcher

FROM scratch

COPY --from=builder /work/loadwatcher /usr/sbin/loadwatcher
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/

USER loadwatcher

ENTRYPOINT ["/usr/sbin/loadwatcher", "-logtostderr"]