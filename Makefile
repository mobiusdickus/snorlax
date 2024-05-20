VERSION = 0.1.0
BUNDLE_IMG = ghcr.io/moon-society/snorlax-operator-bundle:${VERSION}
OPERATOR_IMG = ghcr.io/moon-society/snorlax-operator:${VERSION}
PROXY_IMG = ghcr.io/moon-society/snorlax-proxy:${VERSION}


## Workflows

setup: minikube-delete minikube-start proxy-install operator-crd-install dummy-install
test: minikube-delete minikube-start operator-bundle dummy-install
release-images: proxy-push operator-push


## Local commands

build:
	go build -o snorlax

serve: build
	./snorlax serve

clean:
	rm -f snorlax


## Helm commands

helm-install-local:
	helm install snorlax ./charts/snorlax \
		--create-namespace \
		--namespace snorlax

helm-install-remote:
	helm repo add moon-society https://moon-society.github.io/helm-charts
	helm repo update
	helm install snorlax moon-society/snorlax \
		--create-namespace \
		--namespace snorlax

helm-uninstall:
	helm uninstall snorlax --namespace snorlax

helm-package:
	cd ./charts/snorlax && helm package .
	mv ./charts/snorlax/snorlax-*.tgz .


## Docker commands

proxy-build:
	VERSION=$(VERSION) docker compose build snorlax

proxy-push: proxy-build
	docker push $(PROXY_IMG)

proxy-install: proxy-build
	docker save $(PROXY_IMG) | (eval $$(minikube docker-env) && docker load)

proxy-serve: docker-build
	docker run -p 8080:8080 $(PROXY_IMG) serve


## Minikube commands

minikube-reset: minikube-delete minikube-start

minikube-start:
	minikube start --addons ingress
	kubectl rollout status -n ingress-nginx deployment/ingress-nginx-controller

minikube-delete:
	minikube delete


## Dummy app

dummy-install:
	kubectl apply -f dummy-app/k8s.yaml


## Operator CRD

operator-crd-install:
	cd operator && make install

operator-crd-uninstall:
	cd operator && make uninstall


## Operator

operator-build:
	cd operator && make docker-build IMG=$(OPERATOR_IMG)

operator-push: operator-build
	cd operator && make docker-push IMG=$(OPERATOR_IMG)

operator-deploy:
	cd operator && make deploy IMG=$(OPERATOR_IMG)

operator-helmify:
	cd operator && make helmify
	-rm -rf charts/snorlax
	mv operator/charts/snorlax ./charts/snorlax

operator-run:
	cd operator && make run


## Operator bundle

operator-bundle: operator-bundle-olm-install operator-bundle-init operator-bundle-build operator-bundle-push operator-bundle-deploy

operator-bundle-olm-install:
	operator-sdk olm install

operator-bundle-init:
	cd operator && make bundle

operator-bundle-build:
	cd operator && make bundle-build BUNDLE_IMG=$(BUNDLE_IMG)

operator-bundle-push:
	cd operator && make bundle-push BUNDLE_IMG=$(BUNDLE_IMG)

operator-bundle-deploy:
	cd operator && operator-sdk run bundle $(BUNDLE_IMG)