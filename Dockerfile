FROM golang:1.23-bullseye AS gobuilder

WORKDIR /root

ENV GOOS=linux\
    GOARCH=amd64


COPY go.mod /root/
COPY go.sum /root/
COPY Makefile /root/

RUN go mod download

COPY out_azurelogsingestion/ /root/out_azurelogsingestion/
RUN make build

FROM fluent/fluent-bit:1.9.10-debug

COPY --from=gobuilder /root/out_azurelogsingestion.so /fluent-bit/bin/
COPY fluent-bit.conf /fluent-bit/etc/
COPY plugins.conf /fluent-bit/etc/

EXPOSE 2020

CMD ["/fluent-bit/bin/fluent-bit", "--config", "/fluent-bit/etc/fluent-bit.conf"]
