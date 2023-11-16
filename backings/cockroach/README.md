개요
----

Cockroachdb 클러스터를 구성할 수 있는 도커 이미지(Doker image)를 생성한다.

설계
----

**버전**  
* 도커 : 19.03.12  
* Cockroachdb 이미지 : 20.1.4

**기능**  
- 도커 이미지가 생성되면 모든 Cockroachdb 노드는 서비스를 실행한다.  
- Cockroachdb 는 secure 모드로 실행되며, 인증서의 기본 경로는 "/root/.cockroach-cert"이다.  
- 유저의 아이디와 비밀번호는 compose파일을 통해 전달받는다(아이디의 필드명을 COCKROACH_USER로 사용할 경우 sql 명령어 실행에 문제가 있을 수 있음)  
- 모든 노드는 서비스 실행 시 --join 옵션을 통해 노드 1번으로 조인(Join)한다.  
- 1번 노드는 Cockroachdb 디렉토리(/cockroach/cockroach-data)에 데이터가 있다면 Cockroachdb 서버만 실행한다. 그렇지 않고 데이터가 없다면 클러스터를 초기화한다(cockroach init).
