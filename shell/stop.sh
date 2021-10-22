#!/usr/bin/env bash
ps -ef | grep ./sillygirl | grep -v grep | awk '{print $1}' | xargs kill -9
echo "傻妞程序已经关闭"