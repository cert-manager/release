FROM golang:alpine as builder

RUN go install github.com/cert-manager/release/cmd/cmrel@latest
RUN go install k8s.io/release/cmd/release-notes@v0.7.0

FROM alpine
COPY --from=builder /go/bin/* /usr/local/bin/
