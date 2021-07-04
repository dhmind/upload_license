#!/usr/bin/env bash
set -e 
builds=(
    "uploadlicense darwin amd64 ./"
    "uploadlicense linux amd64 ./"
)

DIST_DIR="./packages"

build(){
    BIN_PATH=$1
    OS=$2
    ARCH=$3
    SRC_PATH=$4
    echo "====================================================================="
    echo "Build started."
    echo "
        OS: ${OS}
        ARCH: ${ARCH}
        BIN_PATH: ${BIN_PATH}
        SRC: ${SRC_PATH}
        "
    BIN_PATH="${DIST_DIR}/${OS}_${ARCH}/${BIN_PATH}"
    mkdir -p ${DIST_DIR}
    
    GOOS=${OS} GOARCH=${ARCH} go build  -o ${BIN_PATH} ${SRC_PATH}

    echo "====================================================================="
    echo "Build completed."
}
for i in "${builds[@]}"; do
    build $i
done
