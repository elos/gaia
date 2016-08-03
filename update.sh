#!/bin/bash

set -e

if [ -e server.pid ]; then
	echo "Killing old server"
	sudo kill $(cat server.pid);
	rm server.pid;
else
	echo "No server running"
fi

if [ -e gaia ]; then
	echo "Removing gaia"
	rm gaia
fi

echo "Downloading gaia"
wget https://github.com/elos/gaia/blob/master/serve/build/linux/gaia?raw=true -O gaia
chmod +x gaia
sudo ./gaia -dbtype=mongo -dbaddr=localhost:27017 -appdir=/home/ubuntu/go/src/github.com/elos/gaia/app -certfile=/etc/letsencrypt/live/elos.pw/fullchain.pem -keyfile=/etc/letsencrypt/live/elos.pw/privkey.pem -port=443 > ~/stdout.txt 2> ~/stderr.txt &
#sudo ./main -dbtype=mongo -dbaddr=localhost:27017 -appdir=/home/ubuntu/go/src/github.com/elos/gaia/app > ~/stdout.txt 2> ~/stderr.txt &
echo "Writing pid file"
echo $! > ~/server.pid
echo "Done"
