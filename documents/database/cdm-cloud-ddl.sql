use cdm;

DROP TABLE IF EXISTS cdm_service_config;
DROP TABLE IF EXISTS cdm_tenant_config;
DROP TABLE IF EXISTS cdm_global_config;
DROP TABLE IF EXISTS cdm_event;
DROP TABLE IF EXISTS cdm_event_error;
DROP TABLE IF EXISTS cdm_event_code_message;
DROP TABLE IF EXISTS cdm_event_code;
DROP TABLE IF EXISTS cdm_schedule;
DROP TABLE IF EXISTS cdm_backup;
DROP TABLE IF EXISTS cdm_user_receive_event;
DROP TABLE IF EXISTS cdm_user_group;
DROP TABLE IF EXISTS cdm_user_role;
DROP TABLE IF EXISTS cdm_user;
DROP TABLE IF EXISTS cdm_group;
DROP TABLE IF EXISTS cdm_role;
DROP TABLE IF EXISTS cdm_tenant_receive_event;
DROP TABLE IF EXISTS cdm_tenant_solution;
DROP TABLE IF EXISTS cdm_tenant;
DROP TABLE IF EXISTS cdm_casbin_rule;


DROP SEQUENCE IF EXISTS cdm_tenant_seq;
DROP SEQUENCE IF EXISTS cdm_role_seq;
DROP SEQUENCE IF EXISTS cdm_group_seq;
DROP SEQUENCE IF EXISTS cdm_user_seq;
DROP SEQUENCE IF EXISTS cdm_event_seq;
DROP SEQUENCE IF EXISTS cdm_backup_seq;
DROP SEQUENCE IF EXISTS cdm_schedule_seq;
DROP SEQUENCE IF EXISTS cdm_event_seq;


-- cdm_tenant
CREATE SEQUENCE IF NOT EXISTS cdm_tenant_seq;
CREATE TABLE IF NOT EXISTS cdm_tenant (
  id INTEGER NOT NULL DEFAULT NEXTVAL ('cdm_tenant_seq') ,
  name VARCHAR(255) NOT NULL ,
  remarks VARCHAR(300) DEFAULT NULL ,
  use_flag BOOLEAN NOT NULL DEFAULT false ,
  created_at INTEGER NOT NULL DEFAULT 0 ,
  updated_at INTEGER NOT NULL DEFAULT 0 ,
  PRIMARY KEY (id)
)  ;
CREATE INDEX IF NOT EXISTS cdm_tenant_name_idx ON cdm_tenant (name);
CREATE INDEX IF NOT EXISTS cdm_tenant_use_flag_idx ON cdm_tenant (use_flag);


-- cdm_tenant_solution
CREATE TABLE IF NOT EXISTS cdm_tenant_solution (
  tenant_id INTEGER NOT NULL ,
  solution VARCHAR(100) NOT NULL ,
  PRIMARY KEY (tenant_id, solution) ,
  CONSTRAINT cdm_tenant_solution_tenant_id_fk FOREIGN KEY (tenant_id) REFERENCES cdm_tenant (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;


-- cdm_tenant_receive_event
CREATE TABLE IF NOT EXISTS cdm_tenant_receive_event (
  code VARCHAR(255) NOT NULL ,
  tenant_id INTEGER NOT NULL ,
  receive_flag BOOLEAN NOT NULL DEFAULT false ,
  PRIMARY KEY (code, tenant_id) ,
  CONSTRAINT cdm_tenant_receive_event_tenant_id_fk FOREIGN KEY (tenant_id) REFERENCES cdm_tenant (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;


-- cdm_role
CREATE SEQUENCE IF NOT EXISTS cdm_role_seq;
CREATE TABLE IF NOT EXISTS cdm_role (
  id INTEGER NOT NULL DEFAULT NEXTVAL ('cdm_role_seq') ,
  solution VARCHAR(100) NOT NULL ,
  role VARCHAR(30) NOT NULL ,
  PRIMARY KEY (id) ,
  CONSTRAINT cdm_role_un UNIQUE (solution, role)
)  ;


-- cdm_group
CREATE SEQUENCE IF NOT EXISTS cdm_group_seq;
CREATE TABLE IF NOT EXISTS cdm_group (
  id INTEGER NOT NULL DEFAULT NEXTVAL ('cdm_group_seq') ,
  tenant_id INTEGER NOT NULL ,
  name VARCHAR(255) NOT NULL ,
  remarks VARCHAR(300) DEFAULT NULL ,
  created_at INTEGER NOT NULL DEFAULT 0 ,
  updated_at INTEGER NOT NULL DEFAULT 0 ,
  deleted_flag BOOLEAN DEFAULT false ,
  PRIMARY KEY (id) ,
  CONSTRAINT cdm_group_tenant_id_fk FOREIGN KEY (tenant_id) REFERENCES cdm_tenant (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;
CREATE INDEX IF NOT EXISTS cdm_group_name_idx ON cdm_group (name);


-- cdm_user
CREATE SEQUENCE IF NOT EXISTS cdm_user_seq;
CREATE TABLE IF NOT EXISTS cdm_user (
  id INTEGER NOT NULL DEFAULT NEXTVAL ('cdm_user_seq') ,
  tenant_id INTEGER NOT NULL ,
  account VARCHAR(30) NOT NULL ,
  name VARCHAR(255) NOT NULL ,
  department VARCHAR(255) DEFAULT NULL ,
  position VARCHAR(255) DEFAULT NULL ,
  email VARCHAR(320) DEFAULT NULL ,
  contact VARCHAR(20) DEFAULT NULL ,
  language_set VARCHAR(30) ,
  timezone VARCHAR(35) ,
  password char(64) NOT NULL ,
  old_password char(64) DEFAULT NULL ,
  password_updated_at INTEGER DEFAULT NULL ,
  password_update_flag BOOLEAN DEFAULT false ,
  last_logged_in_at INTEGER DEFAULT NULL ,
  last_logged_in_ip VARCHAR(40) DEFAULT NULL ,
  last_login_failed_count SMALLINT DEFAULT NULL ,
  last_login_failed_at INTEGER DEFAULT NULL ,
  created_at INTEGER NOT NULL ,
  updated_at INTEGER NOT NULL ,
  PRIMARY KEY (id),
  CONSTRAINT cdm_user_account_un UNIQUE  (account),
  CONSTRAINT cdm_user_email_un UNIQUE  (email),
  CONSTRAINT cdm_user_tenant_id_fk FOREIGN KEY (tenant_id) REFERENCES cdm_tenant (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;
CREATE INDEX IF NOT EXISTS cdm_user_name_idx ON cdm_user (name);
CREATE INDEX IF NOT EXISTS cdm_user_department_idx ON cdm_user (department);


-- cdm_user_role
CREATE TABLE IF NOT EXISTS cdm_user_role (
  user_id INTEGER NOT NULL ,
  role_id INTEGER NOT NULL ,
  PRIMARY KEY (user_id, role_id) ,
  CONSTRAINT cdm_user_role_user_id_fk FOREIGN KEY (user_id) REFERENCES cdm_user (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT cdm_user_role_role_id_fk FOREIGN KEY (role_id) REFERENCES cdm_role (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;


-- cdm_user_group
CREATE TABLE IF NOT EXISTS cdm_user_group (
  user_id INTEGER NOT NULL ,
  group_id INTEGER NOT NULL ,
  PRIMARY KEY (user_id, group_id) ,
  CONSTRAINT cdm_user_group_user_id_fk FOREIGN KEY (user_id) REFERENCES cdm_user (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
  CONSTRAINT cdm_user_group_group_id_fk FOREIGN KEY (group_id) REFERENCES cdm_group (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;


-- cdm_user_receive_event
CREATE TABLE IF NOT EXISTS cdm_user_receive_event (
  code VARCHAR(255) NOT NULL ,
  user_id INTEGER NOT NULL ,
  receive_flag BOOLEAN ,
  PRIMARY KEY (code, user_id) ,
  CONSTRAINT cdm_user_receive_event_user_id_fk FOREIGN KEY (user_id) REFERENCES cdm_user (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;

-- cdm_event_code
CREATE TABLE IF NOT EXISTS cdm_event_code (
  code VARCHAR(255) NOT NULL ,
  solution VARCHAR(100) NOT NULL ,
  level VARCHAR(10) NOT NULL ,
  class_1 VARCHAR(100) NOT NULL ,
  class_2 VARCHAR(100) NOT NULL ,
  class_3 VARCHAR(100) NOT NULL ,
  PRIMARY KEY (code)
)  ;
CREATE INDEX IF NOT EXISTS cdm_event_code_solution ON cdm_event_code (solution);
CREATE INDEX IF NOT EXISTS cdm_event_code_level ON cdm_event_code (level);
CREATE INDEX IF NOT EXISTS cdm_event_code_class ON cdm_event_code (class_1, class_2, class_3);


-- cdm_event_code_message
CREATE TABLE IF NOT EXISTS cdm_event_code_message (
  code VARCHAR(255) NOT NULL ,
  language VARCHAR(30) NOT NULL ,
  brief STRING(255) NOT NULL ,
  detail STRING(1024) NOT NULL ,
  CONSTRAINT cdm_event_code_code_fk FOREIGN KEY (code) REFERENCES cdm_event_code (code) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;

-- cdm_event_error
CREATE TABLE IF NOT EXISTS cdm_event_error (
  code VARCHAR(255) NOT NULL ,
  solution VARCHAR(100) NOT NULL ,
  service VARCHAR(100) NOT NULL ,
  contents VARCHAR(1024) NOT NULL ,
  PRIMARY KEY (code)
)  ;


-- cdm_event
CREATE SEQUENCE IF NOT EXISTS cdm_event_seq;
CREATE TABLE IF NOT EXISTS cdm_event (
  id INTEGER NOT NULL DEFAULT NEXTVAL ('cdm_event_seq') ,
  tenant_id INTEGER NOT NULL ,
  code VARCHAR(255) NOT NULL ,
  error_code VARCHAR(255) ,
  contents TEXT NOT NULL ,
  created_at INTEGER NOT NULL DEFAULT 0 ,
  PRIMARY KEY (id) ,
  CONSTRAINT cdm_event_tenant_id_fk FOREIGN KEY (tenant_id) REFERENCES cdm_tenant (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;
CREATE INDEX IF NOT EXISTS cdm_event_created_at_idx ON cdm_event (created_at);

-- cdm_schedule
CREATE SEQUENCE IF NOT EXISTS cdm_schedule_seq;
CREATE TABLE IF NOT EXISTS cdm_schedule (
  id INTEGER NOT NULL DEFAULT NEXTVAL ('cdm_schedule_seq') ,
  activation_flag BOOLEAN DEFAULT false ,
  topic VARCHAR(255) NOT NULL ,
  message VARCHAR(1024) NOT NULL ,
  start_at INTEGER NOT NULL ,
  end_at INTEGER NOT NULL ,
  type VARCHAR(100) NOT NULL ,
  interval_minute SMALLINT ,
  interval_hour SMALLINT ,
  interval_day SMALLINT ,
  interval_week SMALLINT ,
  interval_month SMALLINT ,
  week_of_month VARCHAR(20) DEFAULT NULL ,
  day_of_month VARCHAR(10) DEFAULT NULL ,
  day_of_week VARCHAR(10) DEFAULT NULL ,
  hour SMALLINT ,
  minute SMALLINT ,
  timezone VARCHAR(35) NOT NULL DEFAULT 'UTC' ,
  PRIMARY KEY (id)
)  ;


-- cdm_backup
CREATE SEQUENCE IF NOT EXISTS cdm_backup_seq;
CREATE TABLE IF NOT EXISTS cdm_backup (
  id INTEGER NOT NULL DEFAULT NEXTVAL ('cdm_backup_seq') ,
  path VARCHAR(4096) NOT NULL ,
  remarks VARCHAR(300) DEFAULT NULL ,
  created_at INTEGER NOT NULL DEFAULT 0 ,
  updated_at INTEGER NOT NULL DEFAULT 0 ,
  PRIMARY KEY (id)
)  ;


-- cdm_global_config
CREATE TABLE IF NOT EXISTS cdm_global_config (
  key VARCHAR(50) NOT NULL ,
  value VARCHAR(1024) NOT NULL ,
  PRIMARY KEY (key)
)  ;


-- cdm_tenant_config
CREATE TABLE IF NOT EXISTS cdm_tenant_config (
  tenant_id INTEGER NOT NULL ,
  key VARCHAR(50) NOT NULL ,
  value VARCHAR(1024) NOT NULL ,
  PRIMARY KEY (tenant_id, key) ,
  CONSTRAINT cdm_tenant_config_tenant_id_fk FOREIGN KEY (tenant_id) REFERENCES cdm_tenant (id) ON UPDATE RESTRICT ON DELETE RESTRICT
)  ;


-- cdm_service_config
CREATE TABLE IF NOT EXISTS cdm_service_config (
  name VARCHAR(50) NOT NULL ,
  key VARCHAR(50) NOT NULL ,
  value VARCHAR(100) NOT NULL ,
  PRIMARY KEY (name, key)
)  ;

 -- cdm_casbin_rule
CREATE TABLE IF NOT EXISTS cdm_casbin_rule (
  p_type      VARCHAR(100),
  v0          VARCHAR(100),
  v1          VARCHAR(100),
  v2          VARCHAR(100),
  v3          VARCHAR(100),
  v4          VARCHAR(100),
  v5          VARCHAR(100)
)  ;
