#/bin/bash

set -e
OPERATION=$1

#DEVICE=eno2
DEVICE=enp0s3


Message() {
  STAMP=$(date +"%y-%m-%dT%H:%M:%S")
  echo "${STAMP} $1"
}

StopProcess() {
  Message "StopProcess"
  set +e
  PID=$(ps -ef | grep httshark | grep device | grep -v grep | awk '{print $2}'| xargs echo)
  if [ "${PID}" != "" ]; then
    sudo kill ${PID}
  fi
  set -e
  sleep 1
}

RunProcess() {
  Message "RunProcess"
  hosts=:54004,:54016,:54010,:54006,:54014,:54007,:50004,:54011,:50001,:54001,:50003,:54013,:54009,:50008,:54003,:54002,:54008,:54005,:50009,:54015,:50002,:50006,:54012,:54017,:54018,:50005,:50007,:50013,:50010,:50011,:50012,:50014,:50015

  echo "sudo ${PWD}/httshark -capture httpdump -device ${DEVICE} -output-folder ${PWD}/output -hosts ${hosts} -har-processors sites-stats -sites-stats-file ${PWD}/logs/sites.csv  >> ${PWD}/logs/httshark.log 2>&1" > /tmp/httshark.sh
  chmod +x /tmp/httshark.sh
  nohup /tmp/httshark.sh >> ./logs/httshark_nohup.log 2>&1 &
}

RestartProcess(){
  Message "RestartProcess"
  StopProcess
  RunProcess
  sleep 20
}

StopWatchdog() {
  Message "StopWatchdog"
  set +e
  PID=$(ps -ef | grep watchdog.sh | grep nohup | awk '{print $2}'| xargs echo)
  if [ "${PID}" != "" ]; then
    sudo kill ${PID}
  fi
  set -e
  sleep 1
}

Cleanup() {
  Message "Cleanup"
  sudo rm -rf output
  sudo mkdir output
  sudo chmod 777 output

  sudo rm -rf logs
  sudo mkdir logs
  sudo chmod 777 logs
}

StartWatchdogNoHop() {
  Message "StartWatchdogNoHop"
  echo "sudo ${PWD}/watchdog.sh start-nohup  >> ${PWD}/logs/watchdog.log 2>&1" > /tmp/watchdog.sh
  chmod +x /tmp/watchdog.sh
  nohup /tmp/watchdog.sh >> ./logs/watchdog_nohup.log 2>&1 &

  Message Started
}

Health() {
  Message "Health"
  curl --fail http://127.0.0.1:6060
  CURL_RC=$?
  echo "curl RC is ${CURL_RC}"
  if [ "${CURL_RC}" != "0" ]; then
    return 1
  fi
}

Watchdog() {
  Message "Watchdog"
  RestartProcess
  while true; do
    set +e
    Health
    HEALTH_RC=$?
    set -e
    if [ "${HEALTH_RC}" != "0" ]; then
      RestartProcess
    fi
    sleep 5
  done
}


if [ "${OPERATION}" = "start" ]; then
  StopWatchdog
  StopProcess
  Cleanup
  StartWatchdogNoHop
elif [ "${OPERATION}" = "start-nohup" ]; then
  Watchdog
elif [ "${OPERATION}" = "stop" ]; then
  StopWatchdog
  StopProcess
else
  echo "Missing operation: start|stop"
fi

