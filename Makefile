version="0.1.1"
version_file=VERSION
working_dir=$(shell pwd)
arch="armhf"

clean:
	-rm ecollector

build-go:
	cd ./src;go build -o ecollector service.go;cd ../

build-go-arm:
	cd ./src;GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o ecollector service.go;cd ../

build-go-amd:
	cd ./src;GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ecollector service.go;cd ../


configure-arm:
	python ./scripts/config_env.py prod $(version) armhf

configure-amd64:
	python ./scripts/config_env.py prod $(version) amd64


package-tar:
	tar cvzf ecollector_$(version).tar.gz ecollector VERSION

package-deb-doc-tp:
	@echo "Packaging application as Thingsplex debian package"
	chmod a+x package/debian_tp/DEBIAN/*
	cp ./src/ecollector package/debian_tp/opt/thingsplex/ecollector
	cp VERSION package/debian_tp/opt/thingsplex/ecollector
	docker run --rm -v ${working_dir}:/build -w /build --name debuild debian dpkg-deb --build package/debian_tp
	@echo "Done"

package-deb-doc-fh:
	@echo "Packaging application as Futurehome debian package"
	chmod a+x package/debian_fh/DEBIAN/*
	cp ./src/ecollector package/debian_fh/usr/bin/ecollector
	cp VERSION package/debian_fh/var/lib/futurehome/ecollector
	docker run --rm -v ${working_dir}:/build -w /build --name debuild debian dpkg-deb --build package/debian_fh
	@echo "Done"


deb-arm-fh : clean configure-arm build-go-arm package-deb-doc-fh
	mv package/debian_fh.deb package/build/ecollector_$(version)_armhf.deb

deb-arm-tp : clean configure-arm build-go-arm package-deb-doc-tp
	mv package/debian_tp.deb package/build/ecollector_$(version)_armhf.deb

deb-amd : configure-amd64 build-go-amd package-deb-doc-tp
	mv debian.deb ecollector_$(version)_amd64.deb

run :
	cd ./src; go run service.go -c testdata/config.json;cd ../


.phony : clean
