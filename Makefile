build-windows:
	set GOOS=windows&&set GOARCH=amd64&&cd src&&set CGO_ENABLED=1&&set CC="x86_64-w64-mingw32-gcc"&&go build -ldflags "-w -s  -H=windowsgui" -o ../bin/helm.exe -mod=readonly

#build-osx:
#	set GOOS=linux&&setGOARCH=amd64&&cd src&&go build -ldflags "-w -s" -o ../bin/helm .

#build-windows:
#	cd src ; env GOOS="windows" GOARCH="amd64" CGO_ENABLED="1" CC="x86_64-w64-mingw32-gcc" fyne package -os windows --exe ../bin/Helm.exe

build-osx:
	cd src ; fyne package -os darwin ; defaults write Helm.app/Contents/Info LSUIElement 1

build-oldosx:
	cd src ; GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 CGO_CFLAGS="-mmacosx-version-min=10.12" CGO_LDFLAGS="-mmacosx-version-min=10.12" go build -mod=readonly

# export GOOS=windows
# export GOARCH=amd64
# export CGO_ENABLED=1
# export CC="x86_64-w64-mingw32-gcc"
# go build -ldflags "-w -s  -H=windowsgui" -o ../bin/helm.exe -mod=readonly