#!/bin/bash

set -eou pipefail
pushd $(dirname "$0")/

terraform destroy -parallelism=5 -force -var-file config.json -var ami_id=
