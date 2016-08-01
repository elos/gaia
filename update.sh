#!/bin/bash

set -e

if [ -e server.pid ]; then
	echo "Killing old server"
	sudo kill $(cat server.pid);
	rm server.pid;
else
	echo "No server running"
fi

echo "Navigating to github.com directory"
cd ~/go/src/github.com/
echo "Removing all elos dirs"
rm -rf elos
echo "Getting gaia"
go get github.com/elos/gaia
echo "Getting models"
go get github.com/elos/models
echo "Going to serve dir"
cd elos/gaia/serve/
echo "Building"
go build main.go
echo "Starting"
sudo ./main -dbtype=mongo -dbaddr=localhost:27017 -appdir=/home/ubuntu/go/src/github.com/elos/gaia/app -certfile=/etc/letsencrypt/live/elos.pw/cert.pem -keyfile=/etc/letsencrypt/live/elos.pw/privkey.pem -port=443 > ~/stdout.txt 2> ~/stderr.txt &
#sudo ./main -dbtype=mongo -dbaddr=localhost:27017 -appdir=/home/ubuntu/go/src/github.com/elos/gaia/app > ~/stdout.txt 2> ~/stderr.txt &
echo "Writing pid file"
echo $! > ~/server.pid
echo "Done"
