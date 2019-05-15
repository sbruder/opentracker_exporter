FROM golang:alpine as builder

WORKDIR /go/src/github.com/sbruder/opentracker_exporter/

COPY opentracker_exporter.go .

RUN apk add --no-cache git upx

RUN go get -v \
    && CGO_ENABLED=0 go build -v -ldflags="-s -w" \
    && upx --ultra-brute opentracker_exporter

FROM scratch

COPY --from=builder /go/src/github.com/sbruder/opentracker_exporter/opentracker_exporter /opentracker_exporter

USER 1000

ENTRYPOINT ["/opentracker_exporter"]

EXPOSE 9574
