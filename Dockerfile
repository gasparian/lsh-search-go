FROM golang:1.13-alpine as builder

RUN mkdir -p "$GOPATH/src/lsh-search-service"
WORKDIR $GOPATH/src/lsh-search-service
COPY . .

# RUN go mod init && \
#     go mod tidy
# RUN go get -v -t ./...
RUN go mod tidy -v

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /usr/bin/app ./main.go

# -----------------------------------------------------------------------------

FROM scratch
COPY --from=builder /usr/bin/app /app
ENTRYPOINT [ "/app" ]
EXPOSE 8080
