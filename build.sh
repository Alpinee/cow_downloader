if [ -z "$1" ]; then
  os="default"
else
  os="$1"
fi

buildWindows() {
    echo "build windows"
    mkdir -p ./bin/windows
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/windows/cow_downloader.exe
}

buildLinux() {
    echo "build linux"
    mkdir -p ./bin/linux
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/linux/cow_downloader
}

buildMac() {
    echo "build mac"
    mkdir -p ./bin/mac
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/mac/cow_downloader
}

# 如果是windows
if [ $os == "windows" ]; then
  buildWindows
  exit 0
fi

# 如果是linux
if [ $os == "linux" ]; then
  buildLinux
  exit 0
fi

# 如果是mac
if [ $os == "mac" ]; then
  buildMac
  exit 0
fi

# 否则都打
buildWindows
buildLinux
buildMac
