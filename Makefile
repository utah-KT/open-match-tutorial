CONTEXT ?= open-match-test
VERSION ?= 1.8.0

change-context:
	kubectl config use-context $(CONTEXT)

install: install-openmatch build apply

install-openmatch: change-context
	helm install -f manifests/open-match-core/values.yaml open-match --create-namespace -n open-match open-match/open-match --version=$(VERSION)

uninstall-openmatch: change-context
	helm uninstall open-match -n open-match

clean: change-context uninstall-openmatch
	kubectl delete namespace open-match-test

apply: change-context
	helm install -f manifests/open-match-tutorial/values.yaml open-match-tutorial --create-namespace -n open-match-test manifests/open-match-tutorial

proto:
	protoc --go_out=./grpc --go-grpc_out=./grpc proto/*.proto

gameserver:
	docker build -t open-match-tutorial-gameserver -f gameserver/Dockerfile .

director:
	docker build -t open-match-tutorial-director -f director/Dockerfile .

mmf:
	docker build -t open-match-tutorial-mmf -f mmf/Dockerfile .

front:
	docker build -t open-match-tutorial-front -f front/Dockerfile .

build: proto gameserver director mmf front

.PHONY: change-context install install-openmatch uninstall-openmatch clean apply proto gameserver director mmf front build
