build-windows:
	set GOOS=windows&&set GOARCH=amd64&&cd src&&set CGO_ENABLED=1&&set CC="x86_64-w64-mingw32-gcc"&&go build -ldflags "-w -s  -H=windowsgui" -o ../bin/hq.exe -mod=readonly
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

#installer: build-osxu
#	cd src
#	[[ -f Helm-Installer.dmg ]] && rm Helm-Installer.dmg
#	mv Helm.app dmgsource
#	create-dmg \
#	--volname "Helm Installer" \
#	--background "../assets/big-seal-arkark.png" \
#	--window-pos 200 120 \
#	--window-size 800 400 \
#	--icon-size 100 \
#	--icon "dmgsource/Helm.app" 200 190 \
#	--hide-extension "Helm.app" \
#	--app-drop-link 600 185 \
#	"Helm-Installer.dmg" \
#	"Helm.app"


# export GOOS=windows
# export GOARCH=amd64
# export CGO_ENABLED=1
# export CC="x86_64-w64-mingw32-gcc"
# go build -ldflags "-w -s  -H=windowsgui" -o ../bin/helm.exe -mod=readonly
