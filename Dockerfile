FROM golang:1.14.3-alpine as build

WORKDIR /go/src/github.com/marcinbudny/eventstore_exporter
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -o app

FROM scratch
COPY --from=build /go/src/github.com/marcinbudny/eventstore_exporter/app /
EXPOSE 9448
ENTRYPOINT [ "/app" ]