FROM golang:1.13-alpine as builder

RUN mkdir -p "$GOPATH/src/github.com/gasparian/lsh-search-service"
WORKDIR $GOPATH/src/github.com/gasparian/lsh-search-service
COPY . .

RUN go get -v -t ./...

# TODO: CGO_ENABLED=0 is that right? Do we need `cgo` param?
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a \
    -installsuffix cgo \
    -ldflags="-w -s" \
    -o /usr/bin/app \
    ./main.go

# -----------------------------------------------------------------------------

FROM scratch
COPY --from=builder /usr/bin/app /app
ENTRYPOINT [ "/app" ]
EXPOSE 8080
