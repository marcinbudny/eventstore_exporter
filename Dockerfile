FROM golang:1.10-alpine as build

WORKDIR /go/src/github.com/marcinbudny/eventstore_exporter
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-extldflags "-static"' -o app

FROM busybox:1.28
COPY --from=build /go/src/github.com/marcinbudny/eventstore_exporter/app /
EXPOSE 9448
ENTRYPOINT [ "/app" ]