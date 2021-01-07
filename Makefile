IMAGE_NAME := "gunstore/cert-manager-webhook-dynu"
IMAGE_TAG := "latest"

OUT := $(shell pwd)/_out

$(shell mkdir -p "$(OUT)")

verify:
	go test -v .

build:
	docker build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: rendered-manifest.yaml
rendered-manifest.yaml:
	helm template \
	    cert-manager-webhook-dynu \
        --set image.repository=$(IMAGE_NAME) \
        --set image.tag=$(IMAGE_TAG) \
        deploy/cert-manager-webhook-dynu > "$(OUT)/rendered-manifest.yaml"

helm-package:
	cd deploy && \
	helm package --version $(IMAGE_TAG) cert-manager-webhook-dynu && \
	cd ..

helm-install:
	helm uninstall cert-manager-webhook-dynu
	helm install cert-manager-webhook-dynu ~/dev/cert-manager-webhook-dynu/deploy/cert-manager-webhook-dynu-$(IMAGE_TAG).tgz