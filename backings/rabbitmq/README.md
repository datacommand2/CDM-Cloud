개요
----

Rabbitmq 클러스터를 구성할 수 있는 도커 이미지(Doker image)를 생성한다.

설계
----

**버전**  
* 도커 : 19.03.12  
* Rabbitmq 이미지 : 3.8.5

**기능**  
도커 이미지가 최초로 생성되면 각 Rabbitmq 노드는 Leader와 Follower로 나뉘어 실행된다.  
- Leader 와 Follower의 역할은 초기화시에만 구분된다.

Rabbitmq 디렉토리(/var/lib/rabbitmq)에 데이터가 있다면 rabbitmq 서버만 실행한다. 그렇지 않다면 각 노드는 역할에 따라 다음과 같은 기능을 수행한다.

-	Leader : 계정을 생성한다.  
-	Follower : Leader에 조인한다.

Rabbitmq의 모든 데이터는 다음 옵션을 통해 모든 노드에 공유된다.

```
{"ha-mode":"all","ha-sync-mode":"automatic"}
```

사용예제
--------

-	Docker swarm 실행

```
# docker stack deploy -c docker-cloud cdm
```
