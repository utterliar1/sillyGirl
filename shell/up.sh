#!/usr/bin/env bash
if [ ! -d "/ql" ];then
  cd /jd/sillygirl && git fetch --all && git reset --hard origin/main && git pull
else
  cd /ql/sillygirl && git fetch --all && git reset --hard origin/main && git pull
fi