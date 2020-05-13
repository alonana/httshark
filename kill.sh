#/bin/bash

ps -ef | grep httshark | grep -v grep | awk '{print $2}' | xargs sudo kill