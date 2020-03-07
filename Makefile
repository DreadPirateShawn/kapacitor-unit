GOLANG_IMAGE=golang:1.12.9-buster

# Mount local project as working directory.
# Run as current host user, to ensure created files have similar permissions.
# (Specify GOCACHE since default /.cache only works as root.)
# Use network host, so that samples can interact with local kapacitor & influxdb.
DOCKER_PARAMS=\
  --mount type=bind,source="$(shell pwd)",target=/kapacitor-unit \
  --workdir /kapacitor-unit \
  --user $(shell stat Makefile --format='%u:%g') \
  --env GOCACHE=/tmp/.cache \
  --network host

travis-ci-setup:
	go get ./cmd/kapacitor-unit ./io ./task ./test

tests:
	docker run -it $(DOCKER_PARAMS) $(GOLANG_IMAGE) \
	  go test -cover ./cmd/kapacitor-unit ./io ./task ./test

build:
	docker run $(DOCKER_PARAMS) $(GOLANG_IMAGE) \
	  go build ./cmd/kapacitor-unit/main.go

start-kapacitor-and-influx:
	docker-compose -f infra/docker-compose.yml up -d

sample1: build
	docker run -it $(DOCKER_PARAMS) $(GOLANG_IMAGE) \
	  ./main -dir ./sample/tick_scripts \
                 -tests ./sample/test_cases/test_case.yaml

sample1_debug: build
	docker run -it $(DOCKER_PARAMS) $(GOLANG_IMAGE) \
	  ./main -dir ./sample/tick_scripts \
	         -tests ./sample/test_cases/test_case.yaml \
	         -stderrthreshold=INFO

sample1_batch: build
	docker run -it $(DOCKER_PARAMS) $(GOLANG_IMAGE) \
	  ./main -dir ./sample/tick_scripts \
	         -tests ./sample/test_cases/test_case_batch.yaml

sample1_batch_debug: build
	docker run -it $(DOCKER_PARAMS) $(GOLANG_IMAGE) \
	  ./main -dir ./sample/tick_scripts \
	         -tests ./sample/test_cases/test_case_batch.yaml \
	         -stderrthreshold=INFO

sample_dir: build
	docker run -it $(DOCKER_PARAMS) $(GOLANG_IMAGE) \
	  ./main -dir ./sample/tick_scripts \
	         -tests ./sample/test_cases
