#!/bin/bash

set -e

make ubuntu
git add ./
git commit -m "releasing gaia [release.sh]"
git push
