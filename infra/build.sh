#!/bin/bash

set -eou pipefail
pushd $(dirname "$0")/..

chmod -rwx infra/files/id_rsa
chmod u+r infra/files/id_rsa

revision=$(git rev-parse ${1:-HEAD})

git archive $revision | gzip > /tmp/talkativebuild.tgz

packercmd="packer build -machine-readable -var git_ref=$revision -var source_tarball=/tmp/talkativebuild.tgz -var-file infra/config.json infra/packer.json"

exec 3>&1
packer_output=$(($packercmd || true) | tee >(cat - >&3))

ami_id=$(echo "$packer_output" | grep -Ei "(name conflicts with an existing ami.*ami-\w{6,})|(artifact,0,id)" | grep -oE "ami-\w{6,}" | tail -1)

echo $ami_id

pushd infra
terraform apply -var-file config.json -var "ami_id=$ami_id"

nodeips=$(terraform output | grep -Eo '([0-9]{1,3}[.]){3}[0-9]{1,3}')
nodehosts=$(echo "$nodeips" | sed 's|^|ubuntu@|')

for nodehost in $nodehosts
do
    while true
    do
        echo waiting for $nodehost
        if echo DOHANGUPPLZ | nc -G 5 -w 2 $(echo $nodehost | cut -d@ -f2) 22
        then
            echo ok
            break
        fi
        sleep 2
    done
    othernodes=$(echo "$nodehosts" | grep -vF $nodehost)
    echo "$(echo $othernodes)" | ssh -o StrictHostKeyChecking=no\
                                     -o UserKnownHostsFile=/dev/null\
                                     -i files/id_rsa\
                                     $nodehost\
                                     -- sudo dd of=/etc/talkativenodes
done
