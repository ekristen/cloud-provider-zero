docs-build:
	docker run --rm -it -p 8000:8000 -v ${PWD}:/docs squidfunk/mkdocs-material build

docs-serve:
	docker run --rm -it -p 8000:8000 -v ${PWD}:/docs squidfunk/mkdocs-material

docs-seed:
	cp README.md docs/index.md

build-kustomize:
	rm kustomize/*.yaml
	helm template cloud-provider-zero chart  --namespace cloud-provider-zero --output-dir kustomize
	mv kustomize/cloud-provider-zero/templates/* kustomize
	rm -r kustomize/cloud-provider-zero
	cd kustomize && kustomize create --autodetect