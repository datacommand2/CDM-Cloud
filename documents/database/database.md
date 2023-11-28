# DataBase

**Table of Contents**

- [SCHEMA](#SCHEMA)
- [ERD](#ERD)

---

## SCHEMA

### 개요

* CDM-CLOUD Database Schema
* ['기능 목록'](../functions.md) 참조하여 작성

### 스크립트

* [DDL 스크립트](cdm-cloud-ddl.sql)
* [DML 스크립트](cdm-cloud-dml.sql)
* [EventCode DML 스크립트](event-code-dml.sql)
* [EventCodeMessage DML 스크립트](event-code-message-dml.sql)
* [EventError DML 스크립트](event-error-dml.sql)
---

## ERD
```plantuml
@startuml
left to right direction
skinparam linetype ortho
hide circle

entity cdm_tenant {
  id : INT <<generated>>
  --
  name : VARCHAR(255) <<NN>> <<index_1>>
  remarks : VARCHAR(300)
  use_flag : TINYINT(1) <<NN>> <<index_2>>
  created_at : INT <<NN>>
  updated_at : INT <<NN>>
}

entity cdm_tenant_solution {
  tenant_id : INT <<FK>>
  solution : VARCHAR(100)
  --
}

entity cdm_tenant_receive_event {
  code : VARCHAR(255) <<FK>>
  tenant_id : INT <<FK>>
  --
  receive_flag : BOOL <<NN>>
}

entity cdm_role {
  id : INT <<generated>>
  --
  solution : VARCHAR(100) <<NN>> <<unique_1>>
  role : VARCHAR(30) <<NN>> <<unique_1>>
}

entity cdm_group {
  id : INT <<generated>>
  --
  tenant_id : INT <<FK>> <<NN>>
  name : VARCHAR(255) <<NN>>
  remarks : VARCHAR(300)
  deleted_flag : BOOL
  created_at : INT <<NN>>
  updated_at : INT <<NN>>
}

entity cdm_user {
  id : INT <<generated>>
  --
  tenant_id : INT <<FK>> <<NN>>
  account : VARCHAR(30) <<NN>> <<unique_2>>
  name : VARCHAR(255) <<NN>> <<index>>
  department : VARCHAR(255) <<index>>
  position : VARCHAR(255)
  email : VARCHAR(50) <<unique_1>>
  contact : VARCHAR(20)
  language_set : VARCHAR(30)
  timezone : VARCHAR(35)
  password : CHAR(64) <<NN>>
  old_password : CHAR(64)
  password_updated_at : INT
  password_update_flag : TINYINT(1)
  last_logged_in_at : INT
  last_logged_in_ip : VARCHAR(40)
  last_login_failed_count : TINYINT(1)
  last_login_failed_at : INT
  created_at : INT <<NN>>
  updated_at : INT <<NN>>
}

entity cdm_user_role {
  user_id : INT <<FK>>
  role_id : INT <<FK>>
  --
}

entity cdm_user_group {
  user_id : INT <<FK>>
  group_id : INT <<FK>>
  --
}

entity cdm_user_receive_event {
  code : VARCHAR(255) <<FK>>
  user_id : INT <<FK>>
  --
  receive_flag : BOOL
}

entity cdm_event {
  id : INT <<generated>>
  --
  tenant_id : INT <<FK>> <<NN>>
  code : VARCHAR(255) <<NN>>
  error_code : VARCHAR(255)
  contents : TEXT <<NN>>
  created_at : INT <<NN>> <<index>>
}

entity cdm_event_code {
  code : VARCHAR(255) <<NN>>
  --
  solution : VARCHAR(100) <<NN>> <<index_1>>
  level : VARCHAR(10) <<NN>> <<index_2>>
  class_1 : VARCHAR(100) <<NN>> <<index_3>>
  class_2 : VARCHAR(100) <<NN>> <<index_3>>
  class_3 : VARCHAR(100) <<NN>> <<index_3>>
}

entity cdm_event_code_message {
  code : VARCHAR(255) <<FK>> <<NN>>
  --
  language : VARCHAR(30) <<NN>>
  brief : STRING(255) <<NN>>
  detail : STRING(1024) <<NN>>
}

entity cdm_event_error {
  code : VARCHAR(255) <<NN>>
  --
  solution : VARCHAR(100) <<NN>>
  service : VARCHAR(100) <<NN>>
  contents : VARCHAR(1024) <<NN>>
}

entity cdm_schedule {
  id : INT <<generated>>
  --
  activation_flag : BOOL
  topic : VARCHAR(255) <<NN>>
  message : VARCHAR(1024) <<NN>>
  start_at : INT <<NN>>
  end_at : INT <<NN>>
  type : VARCHAR(100) <<NN>>
  interval_minute : TINYINT(1)
  interval_hour : TINYINT(1)
  interval_day : TINYINT(1)
  interval_week : TINYINT(1)
  interval_month : TINYINT(1)
  week_of_month : VARCHAR(20)
  day_of_month : VARCHAR(10)
  day_of_week : VARCHAR(10)
  hour : TINYINT(1)
  minute : TINYINT(1)
  timezone : VARCHAR(35) <<NN>>
}

entity cdm_backup {
  id : INT <<generated>>
  --
  path : VARCHAR(4096) <<NN>>
  remarks : VARCHAR(300)
  created_at : INT <<NN>>
  updated_at : INT <<NN>>
}

entity cdm_casbin_rule {
  p_type : VARCHAR(100)
  v0 : VARCHAR(100)
  v1 : VARCHAR(100)
  v2 : VARCHAR(100)
  v3 : VARCHAR(100)
  v4 : VARCHAR(100)
  v5 : VARCHAR(100)
}

entity cdm_global_config {
  key : VARCHAR(50)
  --
  value : VARCHAR(1024) <<NN>>
}

entity cdm_tenant_config {
  tenant_id : INT <<FK>>
  key : VARCHAR(50)
  --
  value : VARCHAR(1024) <<NN>>
}

entity cdm_service_config {
  name : VARCHAR(50)
  key : VARCHAR(50)
  --
  value : VARCHAR(100) <<NN>>
}

cdm_tenant ||-u-o{ cdm_tenant_config
cdm_tenant ||-u-o{ cdm_tenant_solution
cdm_tenant ||-u-o{ cdm_tenant_receive_event
cdm_tenant ||-l-o{ cdm_user
cdm_tenant ||--o{ cdm_group
cdm_tenant ||-r-o{ cdm_event

cdm_event }o-d-|| cdm_event_code
cdm_event_code_message }o-d-|| cdm_event_code
cdm_event }o-d-|| cdm_event_error

cdm_user ||-d-o{ cdm_user_role
cdm_user ||--o{ cdm_user_group
cdm_user ||-l-o{ cdm_user_receive_event

cdm_role ||-l-o{ cdm_user_role
cdm_group ||-l-o{ cdm_user_group

cdm_casbin_rule --[hidden] cdm_backup
cdm_schedule --[hidden] cdm_casbin_rule
cdm_schedule --r[hidden] cdm_global_config
cdm_schedule --r[hidden] cdm_service_config

@enduml
```
