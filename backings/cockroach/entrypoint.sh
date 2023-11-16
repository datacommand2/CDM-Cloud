#!/bin/bash

function create_certification_file() {
  SECURE_PATH=${SECURE_OPTS##*=}
  if [ ! -f  "$SECURE_PATH/ca.crt" ] || [ ! -f  "$SECURE_PATH/ca.key" ]; then
    echo "cockroach occured error starting in secure mode, cause: $SECURE_PATH/ca.key or $SECURE_PATH/ca.crt file not exsits."
    exit -1
  fi

  cockroach cert create-node $HOST 127.0.0.1 localhost --certs-dir=$SECURE_PATH --ca-key=$SECURE_PATH/ca.key

  if [[ $COCKROACH_ROLE == "LEADER" ]]; then
      cockroach cert create-client root --certs-dir=$SECURE_PATH/ --ca-key=$SECURE_PATH/ca.key
  fi
}

function wait_http_ok() {
  COUNT=0
  while true; do
    if [ $COUNT -gt 60 ]; then
      exit 1
    fi

    ret=$(curl -s -o /dev/null -w "%{http_code}" $1)
    if [ $ret -eq 200 ]; then
      break
    fi

    COUNT=$((COUNT+1))
    sleep 1
  done
}

function init_cockroach() {
  # check if leader node
  if [[ $COCKROACH_ROLE != "LEADER" ]]; then
    return 0
  fi

  # check if database initialized
  ret=$(curl -s -o /dev/null -w "%{http_code}" localhost:8080/health?ready=1)
  if [ $ret -eq 200 ]; then
    return 0
  fi

  # initialize cockroach database
  cockroach init $SECURE_OPTS --host=$COCKROACH_LEADER_NODENAME

  if [ $? -ne 0 ]; then
    return 0
  fi

  # wait for database initialized
  wait_http_ok "localhost:8080/health?ready=1"

  # ready check 가 200 이 떨어져도 내부적으로 데이터베이스 초기화가 끝나지 않을 수 있어 sleep 을 추가.
  # 200 이 떨어진 뒤 5초 가 경과해도 데이터베이스 초기화가 끝나지 않았다면 아래 SQL 이 실패하고, entrypoint 가 종료됨
  # 만약 이런 상황이 발생했다면 그 서버 장비는 사용하지 않기를 권장함
  sleep 5

  cockroach sql $SECURE_OPTS -e "create database $COCKROACH_DEFAULT_DATABASE" || exit 1
  cockroach sql $SECURE_OPTS -e "create database $COCKROACH_REPLICATOR_DATABASE"
  cockroach sql $SECURE_OPTS -e "create user $COCKROACH_DEFAULT_USER $LOGIN_OPTS"
  cockroach sql $SECURE_OPTS -e "grant all on database $COCKROACH_DEFAULT_DATABASE to $COCKROACH_DEFAULT_USER"
  cockroach sql $SECURE_OPTS -e "grant all on database $COCKROACH_REPLICATOR_DATABASE to $COCKROACH_DEFAULT_USER"

}

COCKROACH_LEADER_NODENAME="$COCKROACH_LEADER_NODENAME"
DATA_DIR="/cockroach/cockroach-data"
HOST=$HOSTNAME

if [ "$ORCHESTRATION_TOOL" == "kubernetes" ]; then
  DATA_DIR="$DATA_DIR/$HOSTNAME"
  HOST=$(grep $(hostname) /etc/hosts |awk '{print $2}' | head -1)
	DNS=${HOST#"$(hostname)".}
  COCKROACH_LEADER_NODENAME="${HOSTNAME%-*}-0.$DNS"
  ADVERTISE_ADDRESS="--advertise-addr=$HOST:$CDM_SERVICE_ADVERTISE_PORT"
  if [ ${HOSTNAME##*-} -eq 0 ]; then
    COCKROACH_ROLE="LEADER"
  fi
fi

if [ $COCKROACH_INSECURE ] && [ $COCKROACH_INSECURE == true ]
then
  echo "Start insecure mode"
  SECURE_OPTS="--insecure"
  LOGIN_OPTS="LOGIN"
else
  echo "Start secure mode"
  SECURE_OPTS="--certs-dir=/root/.cockroach-cert"
  LOGIN_OPTS="with password $COCKROACH_DEFAULT_PASS"
  create_certification_file
fi

# start cockroach server
cockroach start --join=$COCKROACH_LEADER_NODENAME $SECURE_OPTS $ADVERTISE_ADDRESS --store=$DATA_DIR &

# wait for cockroach server started
wait_http_ok "localhost:8080/health"

sleep 3

# initialize cockroach database
init_cockroach

# register service
/bin/registerd &

sleep infinity