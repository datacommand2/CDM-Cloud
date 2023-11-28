use cdm;

DELETE FROM "cdm_global_config" WHERE 1=1;
DELETE FROM "cdm_tenant_config" WHERE 1=1;
DELETE FROM "cdm_user_role" WHERE 1=1;
DELETE FROM "cdm_role" WHERE 1=1;
DELETE FROM "cdm_group" WHERE 1=1;
DELETE FROM "cdm_user" WHERE 1=1;
DELETE FROM "cdm_tenant_solution" WHERE 1=1;
DELETE FROM "cdm_tenant" WHERE 1=1;


-- cdm_tenant
INSERT INTO cdm_tenant (name, remarks, use_flag, created_at, updated_at) VALUES
('default', 'default tenant', true, 0, 0);

-- cdm_tenant_solution
INSERT INTO cdm_tenant_solution (tenant_id, solution) VALUES
((select id from cdm_tenant where name = 'default'), 'CDM-Cloud');

-- TODO: cdm_role
--       manager 역할은 SaaS 형태로 사용할 경우에만 적재해야 한다.
INSERT INTO cdm_role (solution, role) VALUES
('CDM-Cloud', 'admin'),
('CDM-Cloud', 'manager'),
('CDM-Cloud', 'operator'),
('CDM-Cloud', 'user');

-- cdm_group
INSERT INTO cdm_group (tenant_id, name, remarks, created_at, updated_at) VALUES
((select id from cdm_tenant where name = 'default'), 'default', 'default group', 0, 0);

-- cdm_user
INSERT INTO cdm_user (tenant_id, account, name, password, password_update_flag, language_set, timezone, created_at, updated_at) VALUES
((select id from cdm_tenant where name = 'default'), 'admin', 'administrator', '5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8', true, 'eng', 'UTC', 0, 0);

-- cdm_user_role
INSERT INTO cdm_user_role (user_id, role_id) VALUES
((select id from cdm_user where account = 'admin'), (select id from cdm_role where solution = 'CDM-Cloud' AND role = 'admin'));

-- cdm_tenant_config
INSERT INTO cdm_tenant_config (tenant_id, key, value) VALUES
((select id from cdm_tenant where name = 'default'), 'global_language_set', 'eng'),
((select id from cdm_tenant where name = 'default'), 'global_timezone', 'UTC'),
((select id from cdm_tenant where name = 'default'), 'user_login_restriction_enable', 'true'),
((select id from cdm_tenant where name = 'default'), 'user_login_restriction_try_count', '5'),
((select id from cdm_tenant where name = 'default'), 'user_login_restriction_time', '300'),
((select id from cdm_tenant where name = 'default'), 'user_reuse_old_password', 'false'),
((select id from cdm_tenant where name = 'default'), 'user_password_change_cycle', '90'),
((select id from cdm_tenant where name = 'default'), 'user_session_timeout', '30'),
((select id from cdm_tenant where name = 'default'), 'event_notification_enable', 'false'),
((select id from cdm_tenant where name = 'default'), 'event_email_notification_enable', 'false'),
((select id from cdm_tenant where name = 'default'), 'event_sms_notification_enable', 'false'),
((select id from cdm_tenant where name = 'default'), 'event_desktop_notification_enable', 'false'),
((select id from cdm_tenant where name = 'default'), 'event_popup_notification_enable', 'false'),
((select id from cdm_tenant where name = 'default'), 'event_custom_notification_enable', 'false'),
((select id from cdm_tenant where name = 'default'), 'event_store_period', '12'),
((select id from cdm_tenant where name = 'default'), 'event_smtp_notifier', '{}'),
((select id from cdm_tenant where name = 'default'), 'event_sms_notifier', '{}');

-- cdm_global_config
INSERT INTO cdm_global_config (key, value) VALUES
('global_log_level', 'info'),
('global_log_store_period', '0'),
('system_monitor_interval', '5'),
('backup_schedule_enable', 'false'),
('backup_schedule', '{}'),
('backup_store_period', '12'),
('bug_report_enable', 'false'),
('bug_report_smtp', '{}'),
('bug_report_email', 'admin@datacommand.kr');

-- cdm_casbin_rule
INSERT INTO cdm_casbin_rule (p_type, v0, v1, v2, v3, v4, v5) VALUES
-- cdm-cloud-license
('p', 'manager', 'CDM-Cloud', 'License.GetProductUUID', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'License.SetLicense', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'License.VerifyLicense', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'License.GetLicense', '', '', ''),

-- cdm-cloud-notification
('p', 'user', 'CDM-Cloud', 'Notification.GetEvent', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Notification.GetEvents', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Notification.GetEventsStream', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Notification.GetEventClassifications', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Notification.GetTenantEventReceives', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Notification.SetTenantEventReceives', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Notification.GetUserEventReceives', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Notification.SetUserEventReceives', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Notification.GetConfig', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Notification.SetConfig', '', '', ''),

-- cdm-cloud-monitor
('p', 'admin', 'CDM-Cloud', 'Monitor.GetNodes', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.GetNode', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.GetNodeServices', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.GetServices', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.GetService', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.GetServiceLog', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.GetGlobalConfig', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.SetGlobalConfig', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.GetServiceConfig', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Monitor.SetServiceConfig', '', '', ''),

-- cdm-cloud-identity
('p', 'admin', 'CDM-Cloud', 'Identity.GetTenants', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Identity.AddTenant', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Identity.GetTenant', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Identity.UpdateTenant', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Identity.ActivateTenant', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Identity.DeactivateTenant', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Identity.GetServiceConfigs', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Identity.SetServiceConfig', '', '', ''),
('p', 'admin', 'CDM-Cloud', 'Identity.DeleteServiceConfig', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.AddUser', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.DeleteUser', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.GetUsers', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.ResetUserPassword', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.GetGroups', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.GetGroup', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.AddGroup', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.UpdateGroup', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.DeleteGroup', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.SetGroupUsers', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.GetRoles', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.RevokeSession', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.SetConfig', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.GetConfig', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.GetServiceConfigs', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.SetServiceConfig', '', '', ''),
('p', 'manager', 'CDM-Cloud', 'Identity.DeleteServiceConfig', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Identity.GetUser', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Identity.UpdateUser', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Identity.UpdateUserPassword', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Identity.Login', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Identity.Logout', '', '', ''),
('p', 'user', 'CDM-Cloud', 'Identity.VerifySession', '', '', '')
;

-- TODO: cdm_tenant_receive_event
--       이벤트 코드 정리 후 초기데이터 추가
