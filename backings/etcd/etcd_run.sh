#!/bin/bash

MYHOST=$(hostname)
FINDHOSTS=tasks.$SWARM_SERVICE_NAME
if [ "$ORCHESTRATION_TOOL" == "kubernetes" ]; then
	MYHOST=$(grep $(hostname) /etc/hosts |awk '{print $2}' | head -1)
	ETCD_ID=$((${HOSTNAME##*-}+1))
	FINDHOSTS=${MYHOST#"$(hostname)".}
fi

#ETCD token, name 설정
INITIAL_CLUSTER_TOKEN=$CDM_SERVICE_NAME
ETCD_NAME="$CDM_SERVICE_NAME-$ETCD_ID"

#ETCD restore, data dir 설정 
DATADIR=/data/$ETCD_NAME
SNAPSHOT=$DATADIR/member/snap/db
BACKUP=$DATADIR/backup/db
RESTORE=$DATADIR/restore

[[ ! -d "$DATADIR/backup" ]] && mkdir -p "$DATADIR/backup"
[[ -d $RESTORE ]] && rm -rf $RESTORE
#etcdauth.env 기록된 정보로 auth 확인
[[ ! -z $ETCD_USER && ! -z $ETCD_PASSWORD ]] && etcd_auth="--user $ETCD_USER:$ETCD_PASSWORD"

echo "$(date +%F\ %T) I | Running"
echo "setting initial advertise peer urls and initial cluster"
echo "resolving the container IP with Docker DNS..."

#conatiner 실행 시 할당된 ip 확인
SECONDS=0
while [ -z "$cip" ]; do 
	cip=$(dig +short $MYHOST)
	echo "$cip" | egrep -qe "^[0-9\.]+$"
	if [ -z "$cip" ]; then
  		sleep 1
  		SECONDS=$((SECONDS+1))
	fi
	[[ $SECONDS -gt 10 ]] && break
done

echo "$cip" | egrep -qe "^[0-9\.]+$"
if [ $? -ne 0 ]; then
	echo "warning: unable to resolve this container's IP ($cip), switching back to /etc/hosts"
	cip=$(grep $MYHOST /etc/hosts |awk '{print $1}' | head -1)
	echo "found IP in /etc/hosts: $cip"
else
	echo "resolved IP: $cip"
fi

echo "$cip" | egrep -qe "^[0-9\.]+$"
if [ $? -ne 0 ]; then
	echo "error: unable to get this container's IP ($cip)"
	exit 1
fi

#etcd start 및 restore에 기본 argument 설정 
BASE_ARGS="--name $ETCD_NAME --initial-advertise-peer-urls http://$cip:2380" 
BASE_ARGS="$BASE_ARGS --initial-cluster-token $INITIAL_CLUSTER_TOKEN"

#etcd start에 필요한 argument 설정
START_ARGS="$START_ARGS --listen-peer-urls=http://0.0.0.0:2380"
START_ARGS="$START_ARGS --listen-client-urls=http://0.0.0.0:2379"
START_ARGS="$START_ARGS --auto-compaction-retention=1"
START_ARGS="$START_ARGS --advertise-client-urls=http://${cip}:2379"
START_ARGS="$START_ARGS --data-dir=$DATADIR"
START_ARGS="$START_ARGS --auth-token jwt,pub-key=/bin/.etcd_auth/.jwt_RS256.pub,priv-key=/bin/.etcd_auth/.jwt_RS256,sign-method=RS256"
START_ARGS="$START_ARGS $BASE_ARGS "

#etcd cluster 구성 과정 최초 etcd 데이터 존재 여부 확인
#1. 데이터가 미 존재시
	#1). task slot 확인
		#1-1. task slot이 1번 일 경우
			#A. 자기 노드만 cluster로 구성해서 etcd 실행(new)
			#B. auth 설정 
		#1-2. task slot이 1번이 아닐 경우 
			#A. task 1번 etcd에 자기 노드를 추가 
			#B. etcd 실행(existing)
#2. 데이터가 존재 시
	#1). health check 가능여부 확인
		#(1). health check 가능
			#1-1. member list 정보 획득 가능 여부 확인
				#A. member list 정보 획득 가능
					#a. 현재 member list에 현재 노드 포함 여부 확인
						#a-1). 현재 노드 멤버 포함 
							#a-1-1). etcd_update.sh(백그라운드) 실행 , etcd 실행(existing)
						#a-2). 현재 노드 멤버 미 포함
							#a-1-1). 다른 etcd에 자기 노드를 추가 후 etcd 실행(existing)
				#B. 현재 member list 정보 획득 불 가능
					#a. etcd_update.sh(백그라운드) 실행, etcd 실행(existing)
		#(2). health check 불 가능
			#1-1. task slot 확인
				#A. task slot이 1번일 경우
					#a. 기존 데이터 복구
					#b. 복구 데이터로 자기 노드만 cluster로 구성해서 etcd 실행(new)
				#B. task slot이 1번이 아닐 경우 
					#a. task 1번 etcd에 자기 노드를 추가 
					#b. etcd 실행(existing)

if [ ! -d "$DATADIR/member" ]; then
	echo "etcd dosen't have data. initialzing start"
	if [[ "$ETCD_ID" -eq 1 ]]; then
		echo "etcd node($ETCD_NAME) sets auth"

		START_ARGS="$START_ARGS --initial-cluster $ETCD_NAME=http://$cip:2380 --initial-cluster-state new"
		if [[ ! -z "$etcd_auth" ]] ; then
			/bin/sh /bin/etcd_auth.sh &
		fi
	else
		echo "etcd node($ETCD_NAME) waits to add own node in etcd cluster."
		while :
		do
			sleep 3	
			tips=$(dig +short $FINDHOSTS)
			for tip in $tips; do
				[[ "$cip" = "$tip" ]] && continue
				env=$(etcdctl --endpoints "http://$tip:2379" --command-timeout=2s member add $ETCD_NAME --peer-urls="http://$cip:2380" $etcd_auth | grep ETCD_INITIAL_CLUSTER | tr -d \")
				if [[ $? -eq 0 && ! -z "$env" ]] ; then
					for info in $env 
					do
						eval ${info#*_}
					done
					break
				fi
			done
			[[ ! -z "$INITIAL_CLUSTER" && ! -z "$INITIAL_CLUSTER_STATE" ]] && break
								
		done
		START_ARGS="$START_ARGS --initial-cluster $INITIAL_CLUSTER --initial-cluster-state $INITIAL_CLUSTER_STATE"	
	fi
else
	tips=$(dig +short $FINDHOSTS)
	for tip in $tips; do
		[[ "$cip" = "$tip" ]] && continue
		curl --connect-timeout 5 "http://$tip:2379/health" > /dev/null 2>&1 #
		health=$?
		[[ $health -eq 0 ]] && break
	done
	if [[ "$health" -eq 0 ]]; then
		echo "At least one node is running."
		echo "check etcd node($ETCD_NAME), whether participated in etcd cluster."
		for tip in $tips; do
			[[ "$cip" = "$tip" ]] && continue
			sleep 3
			peers=$(etcdctl --endpoints "http://$tip:2379" --command-timeout=2s member list | cut -d, -f1,2,3,4 | tr -d ' ')
			[[ $? -eq 0 && ! -z "$peers" ]] && break
		done

		if [[ ! -z "$peers"  ]]; then
			for p in $peers ; do
				peerName=$(echo $p | cut -d, -f3)
				if [[ "$peerName" = "$ETCD_NAME" ]];then
					meberexsit=0
					break
				fi  
			done
			if [[ "$meberexsit" -eq 0 ]]; then
				echo "etcd node($ETCD_NAME) had been a part of etcd cluster"
				echo "etcd node($ETCD_NAME) will update own urls to own id"
				/bin/sh /bin/etcd_updatemember.sh "$ETCD_NAME" "http://$cip:2380" "$etcd_auth" &
				START_ARGS="$START_ARGS --initial-cluster-state existing"
			else
				echo "etcd node($ETCD_NAME) waits to add own node in etcd cluster."
				while :
				do
					sleep 3	
					tips=$(dig +short $FINDHOSTS)
					for tip in $tips; do
						[[ "$cip" = "$tip" ]] && continue
						env=$(etcdctl --endpoints "http://$tip:2379" --command-timeout=2s member add $ETCD_NAME --peer-urls="http://$cip:2380" $etcd_auth | grep ETCD_INITIAL_CLUSTER | tr -d \")
						if [[ $? -eq 0 && ! -z "$env" ]] ; then
							for info in $env 
							do
								eval ${info#*_}
							done
							break
						fi
					done
					[[ ! -z "$INITIAL_CLUSTER" && ! -z "$INITIAL_CLUSTER_STATE" ]] && break
									
				done
				rm -rf "$DATADIR/member"
				START_ARGS="$START_ARGS --initial-cluster $INITIAL_CLUSTER --initial-cluster-state $INITIAL_CLUSTER_STATE"
			fi
		else
			echo "etcd node($ETCD_NAME) updates own urls to own id"
			/bin/sh /bin/etcd_updatemember.sh "$ETCD_NAME" "http://$cip:2380" "$etcd_auth" &
			START_ARGS="$START_ARGS --initial-cluster-state existing"
		fi
	else
		echo "etcd cluster was terminated."
		if [[ "$ETCD_ID" -eq 1 ]]; then
			echo "etcd starts restore processing"

			BASE_ARGS="$BASE_ARGS --initial-cluster $ETCD_NAME=http://$cip:2380"
			[[ -f $SNAPSHOT ]] && mv $SNAPSHOT $BACKUP
			
			etcdctl snapshot restore $BACKUP \
			$BASE_ARGS  --data-dir $RESTORE --skip-hash-check=true && \
			rm -rf $DATADIR/member || exit 1

			mv -f $RESTORE/member $DATADIR/ && rm -rf $RESTORE

		else
			echo "etcd node($ETCD_NAME) waits to add own node in etcd cluster."
			while :
			do
				sleep 3	
				tips=$(dig +short $FINDHOSTS)
				for tip in $tips; do
					[[ "$cip" = "$tip" ]] && continue
					env=$(etcdctl --endpoints "http://$tip:2379" --command-timeout=2s member add $ETCD_NAME --peer-urls="http://$cip:2380" $etcd_auth | grep ETCD_INITIAL_CLUSTER | tr -d \")
					if [[ $? -eq 0 && ! -z "$env" ]] ; then
						for info in $env 
						do
							eval ${info#*_}
						done
						break
					fi
				done
				[[ ! -z "$INITIAL_CLUSTER" && ! -z "$INITIAL_CLUSTER_STATE" ]] && break
								
			done
			rm -rf "$DATADIR/member"
			START_ARGS="$START_ARGS --initial-cluster $INITIAL_CLUSTER --initial-cluster-state $INITIAL_CLUSTER_STATE"
		fi
	fi

fi

/bin/registerd & 
exec etcd $START_ARGS
