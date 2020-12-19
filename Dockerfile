FROM golang:1.15-alpine as builder
RUN mkdir /tmp/setup
WORKDIR /tmp/setup
RUN apk add build-base && \
    wget -q ftp://ftp.unidata.ucar.edu/pub/netcdf/netcdf-4/hdf5-1.8.13.tar.gz && \
    tar -xzf hdf5-1.8.13.tar.gz

WORKDIR /tmp/setup/hdf5-1.8.13
RUN ./configure  --prefix=/usr/local && \
    make && make install && \
    rm -rf /tmp/*

RUN mkdir -p "$GOPATH/src/vector-search-go"
WORKDIR $GOPATH/src/vector-search-go
COPY . .

RUN go mod init && \
    go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /usr/bin/app ./main.go

# NOTE: There is a problem compiling with hdf5 for using in scratch image, 
#       so better make it on your local machine with dynamic linking
# RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /usr/bin/bench_data_prep_main ./bench_data_prep_main.go

# -----------------------------------------------------------------------------

FROM scratch
COPY --from=builder /usr/bin/app /app
ENTRYPOINT [ "/app" ]
EXPOSE 8080
