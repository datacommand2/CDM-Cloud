#!/bin/bash

while [[ -z "$process" ]]; do
	process=$(ps | grep -w etcd)
	sleep 1
	COUNT=$((COUNT+1))
	[[ $COUNT -gt 10 ]] && break
done

if [[ -z "$process" ]] ; then
	reboot
fi


etcdctl role get root > /dev/null 2>&1
if [ $? -ne 0 ]; then
	etcdctl role add root > /dev/null 2>&1
	etcdctl role grant-permission --prefix=true root readwrite "/*"  > /dev/null 2>&1
fi
etcdctl user get root > /dev/null 2>&1
if [ $? -ne 0 ]; then
	etcdctl user add root:$ETCD_PASSWORD > /dev/null 2>&1
	etcdctl user grant-role root root > /dev/null 2>&1
fi
etcdctl user get $ETCD_USER > /dev/null 2>&1
if [ $? -ne 0 ]; then
	etcdctl user add $ETCD_USER:$ETCD_PASSWORD > /dev/null 2>&1
	etcdctl user grant-role $ETCD_USER root > /dev/null 2>&1
fi
etcdctl auth enable > /dev/null 2>&1