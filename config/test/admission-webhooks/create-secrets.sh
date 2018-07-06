#!/bin/bash

name=demo-validator.default.svc

if [ ! -s ./snakeoil.key -o ! -s ./snakeoil.crt ]; then
	echo "Creating new snakeoil-secrets..."
	openssl req -newkey rsa:1024 -nodes \
		-days 30 -x509 \
		-subj "/O=Snakeoil Inc/CN=$name" \
		-keyout snakeoil.key -out snakeoil.crt
fi

echo "Injecting snakeoil-secrets into manifests..."
sed -i "s/snakeoil.crt:.*$/snakeoil.crt: `openssl base64 -A < snakeoil.crt`/" \
	admissionvalidator-svc.yaml
sed -i "s/snakeoil.key:.*$/snakeoil.key: `openssl base64 -A < snakeoil.key`/" \
	admissionvalidator-svc.yaml
sed -i "s/caBundle:.*$/caBundle: `openssl base64 -A < snakeoil.crt`/" \
	master-admission-config.yaml

echo "Done."
