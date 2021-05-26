DOWNLOAD=wget -P $(1) -nc $(2)
ANNBENCH_DATA=./test-data
TEST=go test -v -cover $(1) -count=1 -timeout=24h $(2)

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
	$(call TEST,-race,./lsh)
	$(call TEST,-race,./store/...)

.PHONY: annbench
annbench:
	$(call TEST,,./annbench)

install-hdf5:
	sudo apt-get install libhdf5-serial-dev

install-go-deps:
	go get -t -u ./...

install-deps: install-hdf5 install-go-deps
