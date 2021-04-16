DOWNLOAD=wget -P $(1) -nc $(2)
GOBUILD=go fmt $(1); go build -o $(1)/$(2) $(1)
ANNBENCH_DATA=./lsh/annbench/data

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