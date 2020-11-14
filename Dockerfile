FROM golang:1.15-alpine
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

RUN go build -o /usr/bin/app ./main.go && \
    go build -o /usr/bin/prep_bench_data ./prep_bench_data.go

EXPOSE 8080
CMD [ "app" ]
# ENTRYPOINT [ "sh" ]
