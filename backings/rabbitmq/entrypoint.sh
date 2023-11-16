#!/bin/bash

function waitForRabbitStart(){
    while ! rabbitmqctl --node $1 status &> /dev/null
    do
      echo "Wait for the leader node to boot up..."
      sleep 1
    done
}

if [ -z $RABBITMQ_MNESIA_BASE ]; then
  RABBITMQ_MNESIA_BASE="/var/lib/rabbitmq/mnensia"
fi

if [ "$ORCHESTRATION_TOOL" == "kubernetes" ]; then
  MYHOST=$(grep $(hostname) /etc/hosts |awk '{print $2}' | head -1)
  RABBITMQ_NODENAME="rabbit@$MYHOST"
  DNS=${MYHOST#"$(hostname)".}
  RABBITMQ_LEADER_NODENAME="rabbit@${HOSTNAME%-*}-0.$DNS"

  if [ ${HOSTNAME##*-} -eq 0 ]; then
    RABBITMQ_ROLE="LEADER"
  fi

fi

if [ ! -d "$RABBITMQ_MNESIA_BASE" ]
then
  echo "Initialization rabbitmq"
  rabbitmq-server &
  waitForRabbitStart $RABBITMQ_NODENAME
  if [[ $RABBITMQ_ROLE != "LEADER" ]]
  then
    waitForRabbitStart $RABBITMQ_LEADER_NODENAME
    rabbitmqctl stop_app
    rabbitmqctl join_cluster $RABBITMQ_LEADER_NODENAME
    rabbitmqctl start_app
  else
    rabbitmqctl add_user $RABBITMQ_DEFAULT_USER $RABBITMQ_DEFAULT_PASS
    rabbitmqctl set_user_tags $RABBITMQ_DEFAULT_USER administrator
    rabbitmqctl set_permissions $RABBITMQ_DEFAULT_USER ".*" ".*" ".*"
    rabbitmqctl set_policy ha-all "^" '{"ha-mode":"all","ha-sync-mode":"automatic"}'
  fi
else
  echo "Restart Rabbitmq node"
  rabbitmq-server &
fi
/bin/registerd &
sleep infinity
