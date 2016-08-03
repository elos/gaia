#!/bin/bash

set -e

go run main.go --dbtype=mem --port=8080 --seed="./seed.json"
