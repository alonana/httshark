#!/bin/bash

./stop.sh

set -e

sudo rm -rf output
sudo mkdir output
sudo chmod 777 output

sudo rm -rf logs
sudo mkdir logs
sudo chmod 777 logs

instances=0

function run() {
	hosts=$1
	((instances=instances+1))
	sudo ./httshark -device ens192 -output-folder output -hosts ${hosts} > ./logs/${instances}.log 2>&1 &
}
run :54004,:54016,:54010,:54006,:54014 
run :54007,:50004,:54011,:50001,:54001
run :50003,:54013,:54009,:50008,:54003
run :54002,:54008,:54005,:50009,:54015
run :50002,:50006,:54012,:54017,:54018
run :50005,:50007,:50013,:50010,:50011,:50012,:50014,:50015 


echo Started

