#!/bin/bash

set -eou pipefail
pushd $(dirname "$0")/..

revision=$(git rev-parse ${1:-HEAD})
git archive $revision | gzip > /tmp/talkativebuild.tgz

pushd infra

nodeips=$(terraform output | grep -Eo '(\d{1,3}[.]){3}\d{1,3}')
nodehosts=$(echo "$nodeips" | sed 's|^|ubuntu@|')

dossh() {
    ssh -o StrictHostKeyChecking=no\
        -o UserKnownHostsFile=/dev/null\
        -i files/id_rsa\
        $1\
        -- "${@:2}"
}


for nodehost in $nodehosts
do
    cat /tmp/talkativebuild.tgz | dossh $nodehost dd of=/tmp/talkative_stream_test.tgz
    dossh $nodehost mkdir -p /tmp/talkative_stream_test
    dossh $nodehost tar -C /tmp/talkative_stream_test -xzf /tmp/talkative_stream_test.tgz
    dossh $nodehost find /tmp/talkative_stream_test
    dossh $nodehost chmod +x /tmp/talkative_stream_test/infra/files/provision.sh
    dossh $nodehost sudo /tmp/talkative_stream_test/infra/files/provision.sh
done
