#!/bin/bash

set -eou pipefail
pushd $(dirname "$0")/..

revision=$(git rev-parse ${1:-HEAD})

git archive $revision | gzip > /tmp/talkativebuild.tgz

packercmd="packer build -machine-readable -var git_ref=$revision -var source_tarball=/tmp/talkativebuild.tgz -var-file infra/config.json infra/packer.json"

exec 3>&1
packer_output=$(($packercmd || true) | tee >(cat - >&3))

ami_id=$(echo "$packer_output" | grep -Ei "(name conflicts with an existing ami.*ami-\w{6,})|(artifact,0,id)" | grep -oE "ami-\w{6,}" | tail -1)

echo $ami_id

pushd infra
terraform apply -var-file config.json -var "ami_id=$ami_id"

nodeips=$(terraform output | grep -Eo '(\d{1,3}[.]){3}\d{1,3}')

for nodeip in $nodeips
do
    othernodes=$(echo "$nodeips" | grep -vF $nodeip)
    echo "$(echo $othernodes)" | ssh -o StrictHostKeyChecking=no\
                                     -o UserKnownHostsFile=/dev/null\
                                     -i files/id_rsa\
                                     ubuntu@$nodeip\
                                     -- sudo dd of=/etc/talkativenodes
done
