#!/bin/bash

set -euo pipefail

fileroot="/tmp/talkative_stream_test"

dependencies() {
    apt-get update
    apt-get install -y ffmpeg rtmpdump python-minimal
    curl -o /tmp/golang174.tgz https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz
    tar -C /usr/local -xzf /tmp/golang174.tgz
    cat >> /etc/profile <<EOF
export PATH=\$PATH:/usr/local/go/bin
EOF
    export PATH=$PATH:/usr/local/go/bin
}

configssh() {
    cat "$fileroot/infra/files/id_rsa.pub" >> /home/ubuntu/.ssh/authorized_keys
    cp "$fileroot/infra/files/id_rsa" /home/ubuntu/.ssh/id_rsa
}

build() {
    mkdir -p /tmp/goroot/src
    cp -r "$fileroot" /tmp/goroot/src
    pushd /tmp/goroot/src/$(basename "$fileroot")
    GOPATH=/tmp/goroot go build
    mv talkative_stream_test /usr/local/bin/talkativetest
    popd
}

dependencies
configssh
build
