build-windows:
#set GOOS=windows&&set GOARCH=amd64&&cd src&&set CGO_ENABLED=1&&set CC="x86_64-w64-mingw32-gcc"&&go build -ldflags "-w -s  -H=windowsgui" -o ../bin/hq.exe -mod=readonly
	cd src && \
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build -ldflags "-w -s  -H=windowsgui" -o ../bin/hq.exe -mod=readonly
	osslsigncode sign -pkcs12 ~/Dropbox/swap/golang/convergelookup/fyne-cross/dist/windows-arm64/codesign.pfx -pass "nopasswordforyou" -t http://timestamp.digicert.com -in hq.exe -out signed-hq.exe

build-osx:
	cd src && \
	fyne package -os darwin && \
	defaults write HQ.app/Contents/Info LSUIElement 1

build-oldosx:
	cd src && \
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 CGO_CFLAGS="-mmacosx-version-min=10.12" CGO_LDFLAGS="-mmacosx-version-min=10.12" go build -mod=readonly -o ../bin/macold

build-newosx:
	cd src && \
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 CGO_CFLAGS="-mmacosx-version-min=10.14" CGO_LDFLAGS="-mmacosx-version-min=10.14" go build -mod=readonly -o ../bin/macnew

build-osxu: build-oldosx build-newosx build-osx
	cd bin && \
	lipo -create -output hq macold macnew && \
	cp hq ../src/HQ.app/Contents/MacOS && \
	codesign -f -s - ../src/HQ.app