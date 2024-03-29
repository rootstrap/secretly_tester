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

build() {
    rm -rf /tmp/goroot/src/github.com/toptier
    mkdir -p /tmp/goroot/src/github.com/toptier
    cp -r "$fileroot" /tmp/goroot/src/github.com/toptier/secretly_tester
    pushd /tmp/goroot/src/github.com/toptier/secretly_tester
    pwd
    find .
    GOPATH=/tmp/goroot go get
    GOPATH=/tmp/goroot go build -o talkative_stream_test
    mv 640.flv /home/ubuntu/
    mv talkative_stream_test /usr/local/bin/talkative_stream_test
    mv display.py /usr/local/bin/talkative_display.py
    mv infra/files/runtest /usr/local/bin/talkative_runtest
    mv infra/files/autoshutdown.sh /usr/local/bin/
    mv infra/files/autoshutdown.service /etc/systemd/system/
    chmod +x /usr/local/bin/talkative_display.py\
             /usr/local/bin/talkative_runtest\
             /usr/local/bin/autoshutdown.sh
    systemctl enable autoshutdown
    popd
}

dependencies
build
