build-windows:
	set GOOS=windows&&set GOARCH=amd64&&cd src&&set CGO_ENABLED=1&&set CC="x86_64-w64-mingw32-gcc"&&go build -ldflags "-w -s  -H=windowsgui" -o ../bin/helm.exe

#build-osx:
#	set GOOS=linux&&setGOARCH=amd64&&cd src&&go build -ldflags "-w -s" -o ../bin/helm .

#build-windows:
#	cd src ; env GOOS="windows" GOARCH="amd64" CGO_ENABLED="1" CC="x86_64-w64-mingw32-gcc" fyne package -os windows --exe ../bin/Helm.exe

build-osx:
	cd src ; fyne package -os darwin ; defaults write Helm.app/Contents/Info LSUIElement 1
