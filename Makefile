IMAGE_NAME := gunstore/cert-manager-webhook-dynu
IMAGE_TAG := "1.1.23"

OUT := $(shell pwd)/_out

$(shell mkdir -p "$(OUT)")

verify:
  # replace {DYNU_APIKEY} in config.json with env var value
	sed -e 's/{DYNU_APIKEY}/${DYNU_APIKEY}/' testdata/config.json.tpl > testdata/config.json
	# replace {DYNU_APIKEY_B64} in secret-dynu-credentials.yaml with env var value
	sed -e 's/{DYNU_APIKEY_B64}/${DYNU_APIKEY_B64}/' testdata/secret-dynu-credentials.yaml.tpl > testdata/secret-dynu-credentials.yaml
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
	sed -e 's|{IMAGE_NAME}|${IMAGE_NAME}|;s|{IMAGE_TAG}|${IMAGE_TAG}|' deploy/cert-manager-webhook-dynu/values.yaml.tpl > deploy/cert-manager-webhook-dynu/values.yaml
	cd deploy && \
	helm package --version $(IMAGE_TAG) cert-manager-webhook-dynu && \
	cd ..

helm-install:
	-helm uninstall cert-manager-webhook-dynu
	helm install cert-manager-webhook-dynu ~/dev/cert-manager-webhook-dynu/deploy/cert-manager-webhook-dynu-$(IMAGE_TAG).tgz

deploy: build helm-package helm-install
	    --name cert-manager-webhook-dynu \
        --set image.repository=$(IMAGE_NAME) \
        --set image.tag=$(IMAGE_TAG) \
        deploy/cert-manager-webhook-dynu > "$(OUT)/rendered-manifest.yaml"
