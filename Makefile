version="0.8.6"
#version:=$(shell git describe --tags --always | cut -c 2-)
version_file=VERSION
working_dir=$(shell pwd)
arch="armhf"
remote_host = "fh@cube.local"
reprepo_host = "reprepro@archive.futurehome.no"

clean:
	-rm src/ecollector

build-go:
	cd ./src;go build -o ecollector service.go;cd ../

build-go-arm:
	cd ./src;GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w -X main.Version=${version}" -o ecollector service.go;cd ../

build-go-linux-amd64:
	cd ./src;GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${version}" -o ecollector service.go;cd ../

build-go-mac-amd64:
	cd ./src;GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${version}" -o ecollector service.go;cd ../

build-go-win-amd64:
	cd ./src;GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${version}" -o ecollector service.go;cd ../

configure-arm:
	python ./scripts/config_env.py prod $(version) armhf

configure-amd64:
	python ./scripts/config_env.py prod $(version) amd64

run-influxdb-1711:
	docker run --name influxdb1 -d -p 8086:8086 -v influxdb:/var/lib/influxdb influxdb:1.7.11-alpine

run-influxdb-211:
	docker run --name influxdb2 -d -p 8087:8086 -v influxdb2:/var/lib/influxdb2 influxdb:2.1.1-alpine

package-tar:
	cp ./src/ecollector package/tar/
	tar cvzf ecollector_$(version).tar.gz package/tar

package-deb-doc-tp:
	@echo "Packaging application as Thingsplex debian package"
	chmod a+x package/debian_tp/DEBIAN/*
	mkdir -p package/build
	cp ./src/ecollector package/debian_tp/opt/thingsplex/ecollector
	cp VERSION package/debian_tp/opt/thingsplex/ecollector
	docker run --rm -v ${working_dir}:/build -w /build --name debuild debian dpkg-deb --build package/debian_tp
	@echo "Done"

package-deb-doc-tp-linux:
	@echo "Packaging application as Thingsplex debian package"
	chmod a+x package/debian_tp/DEBIAN/*
	mkdir -p package/build
	cp ./src/ecollector package/debian_tp/opt/thingsplex/ecollector
	cp VERSION package/debian_tp/opt/thingsplex/ecollector
	dpkg-deb --build package/debian_tp
	@echo "Done"

deb-arm : clean configure-arm build-go-arm package-deb-doc-tp
	mv package/debian_tp.deb package/build/ecollector_$(version)_armhf.deb

deb-arm-linux : clean configure-arm build-go-arm package-deb-doc-tp-linux
	mv package/debian_tp.deb package/build/ecollector_$(version)_armhf.deb

deb-amd : configure-amd64 build-go-linux-amd64 package-deb-doc-tp
	mv debian.deb ecollector_$(version)_amd64.deb

tar-mac-amd64: clean build-go-mac-amd64 package-tar
	@echo "MAC-amd64 application was packaged into tar archive "

tar-win-amd64: clean build-go-win-amd64 package-tar
	@echo "MAC-amd64 application was packaged into tar archive "

tar-linux-amd64: clean build-go-linux-amd64 package-tar
	@echo "MAC-amd64 application was packaged into tar archive "

build-go-amd64:
	cd ./src;GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${version}" -mod=vendor -o ../package/docker/build/amd64/ecollector service.go;cd ../

build-go-arm64:
	cd ./src;GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.Version=${version}" -mod=vendor -o ../package/docker/build/arm64/ecollector service.go;cd ../

docker-amd64 : build-go-amd64 package-docker-amd64

docker-arm64 : build-go-arm64 package-docker-arm64

package-docker-local: build-go-amd64
	docker build --build-arg TARGETARCH=amd64 -t thingsplex/ecollector:${version} -t thingsplex/ecollector:latest ./package/docker

package-docker-multi:
	docker buildx build --platform linux/arm64,linux/amd64 -t thingsplex/ecollector:${version} -t thingsplex/ecollector:latest ./package/docker --push

docker-multi-setup : configure-amd64
	docker buildx create --name mybuilder
	docker buildx use mybuilder

docker-multi-publish : build-go-arm64 build-go-amd64 package-docker-multi

run :
	cd ./src; go run service.go -c testdata/config.json;cd ../

upload :
	scp package/build/ecollector_$(version)_armhf.deb $(remote_host):~/

upload-install : upload
	ssh -t $(remote_host) "sudo dpkg -i ecollector_$(version)_armhf.deb"

remote-install : deb-arm upload
	ssh -t $(remote_host) "sudo dpkg -i ecollector_$(version)_armhf.deb"

publish-reprepo:
	scp package/build/ecollector_$(version)_armhf.deb $(reprepo_host):~/apps

run-docker:
	docker run -d -v ecollector:/thingsplex/ecollector/data -e MQTT_URI=tcp://192.168.86.33:1884 -e MQTT_USERNAME=shurik -e MQTT_PASSWORD=molodec --network host --name ecollector thingsplex/ecollector:latest

print-version :
	@echo $(version)

.phony : clean
