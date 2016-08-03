#!/bin/bash

set -e

#make ubuntu
git add --all
git commit -m "releasing gaia [release.sh]"
git push
