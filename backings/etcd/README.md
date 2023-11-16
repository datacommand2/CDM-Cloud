## cdm-cloud-etcd
  * cdm_cloud에서 사용하는 kvstore etcd 이미지
    * quay.io/coreos/etcd:v3.3.22 를 기반으로 함

## 구성 
  * etcd_run.sh
    * cdm-cloud-etcd 이미지에 entrypint
    * etcd 클러스터 구성 및 etcd 시작 스크립트
  * etcd_auth.sh
    * 환경 변수로 설정 된 정보로 auth 설정 스크립트
  * etcd_updatemember.sh
    * etcd 클러스터의 member update 스크립트

## 제약사항
  * 환경 변수 값 설정
    * CDM_SERVICE_NAME: service register에 필요한 서비스 명
    * ETCD_ID: etcd_run.sh에서 사용하는 task.slot
    * ETCD_USER: etcd auth 를 설정하기 위한 user명 
    * ETCD_PASSWORD: etcd auth 를 설정하기 위한 password (root, ETCD_USER )