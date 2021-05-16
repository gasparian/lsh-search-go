DOWNLOAD=wget -P $(1) -nc $(2)
ANNBENCH_DATA=./test-data

.SILENT:

download-annbench-data:
	mkdir -p $(ANNBENCH_DATA)
	echo "=== Downloading fashion mnist dataset... ==="
	$(call DOWNLOAD,$(ANNBENCH_DATA),http://ann-benchmarks.com/fashion-mnist-784-euclidean.hdf5)
	echo "=== Downloading NY times dataset... ==="
	$(call DOWNLOAD,$(ANNBENCH_DATA),http://ann-benchmarks.com/nytimes-256-angular.hdf5)
	echo "=== Downloading complete ==="	

.ONESHELL:
.SHELLFLAGS=-e -c

test:
	path=$(path)
	if [ -z "$$path" ]
	then
	    path=./...
	fi
	go test -v -cover -race -timeout=3600s -count=1 $(path)

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
	go get -t -u ./...

install-deps: install-hdf5 install-go-deps
