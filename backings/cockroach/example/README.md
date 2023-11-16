개요
----

이 문서는 cockroach DB의 인증서 생성과 docker swarm secret 등록을 설명하는 문서입니다.

상세
----

-	이 문서는 [Cockroach DB 공식문서](https://www.cockroachlabs.com/docs/stable/orchestrate-cockroachdb-with-docker-swarm.html)에서 인증서를 등록하는 부분만 정리한 것입니다.  
-	인증서 생성은 cockroach 에서 제공하는 기능을 사용(cockroach cert)를 이용합니다(OpenSSL은 추후 추가할 예정).
-	CA 파일은 <u>(주)데이타 커맨드</u>의 CA를 사용합니다.
-	인증서는 docker swarm에 경우 docker secret create 명령을 이용해 도커 이미지 내에서 사용하며, kubernetes에 경우 yaml 파일에 정의 하였습니다.
	- 도커 이미지 내의 인증서 기본 경로는 "/root/.cockroach-cert" 입니다.

키 생성 및 등록
---------------

```
#사용 방법
$ cockroach cert create-ca --certs-dir=<인증서 디렉토리> --ca-key=<키 디렉토리>/ca.key

#예시
$ cockroach cert create-ca --certs-dir=$PWD/.cockroach-cert/ca --ca-key=$PWD/.cockroach-cert/ca/ca.key
```

-	Secret 등록

```
#사용 방법
$ docker secret create <인증서이름> <인증서 디렉토리>/ca.crt
$ docker secret create <인증서이름> <인증서 디렉토리>/ca.key

#예시
$ docker secret create cdm-cloud-ca.crt $PWD/.cockroach-cert/ca/ca.crt
$ docker secret create cdm-cloud-ca.crt $PWD/.cockroach-cert/ca/ca.key
```