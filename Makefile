DOWNLOAD=wget -P $(1) -nc $(2)
GOBUILD=go fmt $(1); go build -o $(1)/$(2) $(1)
PWD=$(shell pwd)
ANNBENCH_DATA=./test-data

.SILENT:

build: 
	$(call GOBUILD,./,lsh-app)

run:
	./lsh-app

download-annbench-data:
	mkdir -p $(ANNBENCH_DATA)
	echo "=== Downloading fashion mnist dataset... ==="
	$(call DOWNLOAD,$(ANNBENCH_DATA),http://ann-benchmarks.com/fashion-mnist-784-euclidean.hdf5)
	echo "=== Downloading lastfm dataset... ==="
	$(call DOWNLOAD,$(ANNBENCH_DATA),http://ann-benchmarks.com/lastfm-64-dot.hdf5)
	echo "=== Downloading complete ==="

.ONESHELL:
.SHELLFLAGS=-e -c

test:
	path=$(path)
	if [ -z "$$path" ]
	then
	    path=./...
	fi
	go clean -testcache
	go test -v -cover -coverprofile cover.out -race ./...

install-hdf5:
	mkdir -p /tmp/hdf5 && cd /tmp/hdf5
	sudo apt-get install build-essential
	wget -q ftp://ftp.unidata.ucar.edu/pub/netcdf/netcdf-4/hdf5-1.8.13.tar.gz
	tar -xzf hdf5-1.8.13.tar.gz
	cd /tmp/hdf5/hdf5-1.8.13
	./configure  --prefix=/usr/local
	make
	sudo make install
	rm -rf /tmp/hdf5/

install-go-deps:
	go get -t 

install-deps: install-hdf5 install-go-deps