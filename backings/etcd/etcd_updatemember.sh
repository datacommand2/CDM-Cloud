#!/bin/bash

COUNT=0
while [[ -z "$process" ]]; do
	process=$(ps | grep -w etcd)
	sleep 1
	COUNT=$((COUNT+1))
	[[ $COUNT -gt 10 ]] && break
done

if [[ -z "$process" ]] ; then
	reboot
fi
COUNT=0
while [[ -z "$peers" ]]; do
	peers=$(etcdctl --command-timeout=2s member list | cut -d, -f1,2,3,4 | tr -d ' ')
	for p in $peers; do
		peerName=$(echo $p | cut -d, -f3)
		peerID=$(echo $p | cut -d, -f1)
		peerUrl=$(echo $p | cut -d, -f4)
		if [[ "$peerName" == $1 && "$peerUrl" != "$2" ]] ;then 
			echo "etcd updates member success"
			etcdctl member update "$peerID" --peer-urls=$2 $3
			break
		fi
	done
	sleep 2 
	COUNT=$((COUNT+1))
	[[ $COUNT -gt 10 ]] && break
done

if [[ $COUNT -gt 10 ]]; then
	echo "etcd retries to update member in all node"
	tips=$(dig +short tasks.$CDM_SERVICE_NAME)
	for tip in $tips; do
		[[ "$cip" = "$tip" ]] && continue
		peers=$(etcdctl --endpoints="http://$tip:2379" --command-timeout=2s member list | cut -d, -f1,2,3,4 | tr -d ' ')
		for p in $peers; do
			peerName=$(echo $p | cut -d, -f3)
			peerID=$(echo $p | cut -d, -f1)
			peerUrl=$(echo $p | cut -d, -f4)
			if [[ "$peerName" == $1 && "$peerUrl" != "$2" ]] ;then 
				echo "etcd updates member success"
				SUCCESS=0
				etcdctl --endpoints="http://$tip:2379" member update "$peerID" --peer-urls=$2 $3
				exit
			fi
		done
	done
	if [[ $SUCCESS -ne 0 ]]; then
		reboot
	fi
fi