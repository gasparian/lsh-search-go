DOWNLOAD=wget -P $(1) -nc $(2)
GOBUILD=go fmt $(1); go build -o $(1)/$(2) $(1)
ANNBENCH_DATA=./annbench/preprocessing/data

.SILENT:

build: 
	$(call GOBUILD,./,lsh-app)

build-annbench: 
	$(call GOBUILD,./annbench/bench,annbench)
	$(call GOBUILD,./annbench/preprocessing,annbench-prep)

build-all: build build-annbench

run:
	./lsh-app

run-annbench:
	./annbench/bench/annbench

run-annbench-prep:
	./annbench/preprocessing/annbench-prep

test-coverage:
	go tool cover -func cover.out | grep total | awk '{print $$3}'

docker-build:
	docker build -t lsh-search-service:latest .

docker-run:
	docker run 
		--rm -it \
		-p 8080:8080 \
		--cpus 4 \
		-m 4096m \
		--env-file config.env \
		lsh-search-service:latest

download-bench-data:
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
	go test -v -cover -coverprofile cover.out -race $$path