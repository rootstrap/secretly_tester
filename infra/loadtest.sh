#!/bin/bash

set -eou pipefail
pushd "$(dirname "$0")/.." > /dev/null
rootdir="$(pwd)"
popd > /dev/null

main() {
    if [[ $# -ne 1 ]]
    then
        usage
    fi

    case $1 in
    build)
        build
        ;;
    ssh)
        randssh
        ;;
    destroy)
        destroy
        ;;
    *)
        usage
        ;;
    esac
}

usage() {
    cat <<EOF >&2
Usage:

    $0 build          build test infrastructure
    $0 ssh            ssh into a random instance
    $0 destroy        destroy test infrastructure
EOF
    exit 2
}

build() {
    cd "$rootdir"
    revision=$(git rev-parse ${1:-HEAD})

    git archive $revision | gzip > /tmp/talkativebuild.tgz

    packercmd="packer build -machine-readable -var git_ref=$revision -var source_tarball=/tmp/talkativebuild.tgz -var-file infra/config.json infra/packer.json"

    exec 3>&1
    packer_output=$(($packercmd || true) | tee >(cat - >&3))

    ami_id=$(echo "$packer_output" | grep -Ei "(name conflicts with an existing ami.*ami-\w{6,})|(artifact,0,id)" | grep -oE "ami-\w{6,}" | tail -1)

    echo $ami_id

    cd infra
    terraform apply -parallelism=5 -var-file config.json -var "ami_id=$ami_id"

    nodeips="$(nodeips)"
    for nodeip in $nodeips
    do
        success=1
        othernodes=$(echo "$nodeips" | grep -vF $nodeip | sed 's|^|ubuntu@|')
        while [[ "$success" -ne "0" ]]
        do
            set +e
            echo $othernodes | dossh ubuntu@$nodeip -- sudo dd of=/etc/talkativenodes
            success="$?"
            set -e
            sleep 2
        done
    done
}

randssh() {
    randomip=$(nodeips | while read -r ip; do echo $RANDOM $ip; done | sort -n | head -1 | cut -d' ' -f2)
    dossh "ubuntu@$randomip" "$@"
}

destroy() {
    cd "$rootdir/infra"
    terraform destroy -parallelism=5 -force -var-file config.json -var ami_id=
}

nodeips() {
    pushd "$rootdir/infra" > /dev/null
    set +e
    terraform output | grep -Eo '([0-9]{1,3}[.]){3}[0-9]{1,3}'
    success=$?
    set -e
    popd > /dev/null
    return $success
}

dossh() {
    ssh -o StrictHostKeyChecking=no\
        -o UserKnownHostsFile=/dev/null\
        -o ConnectTimeout=5\
        -o ConnectionAttempts=3\
        -i "$(getidrsapath)"\
        "$@"
}

getidrsapath() {
    idrsapath="$rootdir/infra/files/id_rsa"
    chmod -rwx "$idrsapath"
    chmod u+r "$idrsapath"
    echo "$idrsapath"
}

main "$@"
