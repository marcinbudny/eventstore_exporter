FROM golang:1.18-alpine as build

WORKDIR /go/src/github.com/marcinbudny/eventstore_exporter
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -o app

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=build /go/src/github.com/marcinbudny/eventstore_exporter/app /
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 9448
ENTRYPOINT [ "/app" ]