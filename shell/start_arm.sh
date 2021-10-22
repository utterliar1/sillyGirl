#!/usr/bin/env bash
if [ ! -d "/ql" ];then
  cd /jd/sillygirl && nohup ./sillygirl_arm64 > xdd.txt 2>&1 &
  echo "傻妞程序正在启动中，请稍后。。。"
else
  cd /ql/sillygirl && nohup ./sillygirl_arm64 > xdd.txt 2>&1 &
  echo "傻妞程序正在启动中，请稍后。。。"
fi
