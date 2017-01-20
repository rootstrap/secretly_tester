#!/bin/bash

set -eou pipefail
pushd $(dirname "$0")/

terraform destroy -force -var-file config.json -var ami_id=
