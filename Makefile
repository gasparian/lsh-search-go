DOWNLOAD=wget -P $(1) -nc $(2)
GOBUILD=go fmt $(1); go build -o $(1)/$(2) $(1)
PWD=$(shell pwd)
ANNBENCH_DATA=./test-data

.SILENT:

build-lsh: 
	cd ./lsh && $(call GOBUILD,./,lsh-app)

run-lsh:
	./lsh/lsh-app

# build-all: build-lsh

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
	cd $$path && go test -v -cover -coverprofile cover.out -race ./...

install-hdf5:
	# sudo su
	mkdir -p /tmp/hdf5 && cd /tmp/hdf5
	# apt-get install build-essential
	wget -q ftp://ftp.unidata.ucar.edu/pub/netcdf/netcdf-4/hdf5-1.8.13.tar.gz
	tar -xzf hdf5-1.8.13.tar.gz
	cd /tmp/hdf5/hdf5-1.8.13
	./configure  --prefix=/usr/local
	make && make install
	rm -rf /tmp/hdf5/

install-go-deps:
	for d in */ ; do
		cd $(PWD)/$$d
		go get -t 2>/dev/null || true
	    echo 1>/dev/null 2>&1
	done

install-deps: install-hdf5 install-go-deps