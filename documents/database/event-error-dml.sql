use cdm;

DELETE FROM cdm_event_error WHERE 1=1;

-- CDM-Cloud.Common
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('required_parameter', 'CDM-Cloud', 'Common', 'detail: required parameter \n parameter: <%= param %>'),
('unchangeable_parameter', 'CDM-Cloud', 'Common', 'detail: unchangeable parameter \n parameter: <%= param %>'),
('conflict_parameter', 'CDM-Cloud', 'Common', 'detail: conflict parameter value \n parameter: <%= param %> \n value: <%= value %>'),
('invalid_parameter', 'CDM-Cloud', 'Common', 'detail: invalid parameter value \n parameter: <%= param %> \n value: <%= value %> \n cause: <%= cause %>'),
('length_overflow_parameter', 'CDM-Cloud', 'Common', 'detail: length overflow parameter value \n parameter: <%= param %> \n value: <%= value %> \n maximum length: <%= max_length %>'),
('out_of_range_parameter', 'CDM-Cloud', 'Common', 'detail: out of range parameter value \n parameter: <%= param %> \n value: <%= value %> \n minimum: <%= min %> \n maximum: <%= max %>'),
('unavailable_parameter', 'CDM-Cloud', 'Common', 'detail: unavailable parameter value \n parameter: <%= param %> \n value: <%= value %> \n available values: <%= available_values %>'),
('pattern_mismatch_parameter_value', 'CDM-Cloud', 'Common', 'detail: pattern mismatch parameter value \n parameter: <%= param %> \n value: <%= value %> \n pattern: <%= pattern %>'),
('format_mismatch_parameter', 'CDM-Cloud', 'Common', 'detail: format mismatch parameter value \n parameter: <%= param %> \n value: <%= value %> \n format: <%= format %>'),
('invalid_request', 'CDM-Cloud', 'Common', 'detail: invalid request \n client ip: <%= client_ip %>'),
('unauthenticated_request', 'CDM-Cloud', 'Common', 'detail: unauthenticated request \n client ip: <%= client_ip %> \n session key: <%= session_key %>'),
('unauthorized_request', 'CDM-Cloud', 'Common', 'detail: unauthorized request \n client ip: <%= client_ip %> \n user name: <%= user_name %> \n user account: <%= user_account %>'),
('no_content', 'CDM-Cloud', 'Common', 'detail: no content'),
('unusable_database', 'CDM-Cloud', 'Common', 'detail: unusable database \n cause: <%= cause %> \n database error: <%= db_error %>'),
('unusable_store', 'CDM-Cloud', 'Common', 'detail: unusable key-value store \n cause: <%= cause %>'),
('unusable_broker', 'CDM-Cloud', 'Common', 'detail: unusable broker \n cause: <%= cause %>'),
('unknown', 'CDM-Cloud', 'Common', 'detail: unknown error \n cause: <%= cause %>'), -- common, center-manager 에서 사용
('ipc_failed', 'CDM-Cloud', 'Common', 'detail: inter process communication failed \n error code: <%= code %> \n message: <%= message %>'),
('not_found_tenant', 'CDM-Cloud', 'Common', 'detail: not found tenant \n tenant ID: <%= id %>'), -- common, identity, notification 에서 사용
('not_found_key', 'CDM-Cloud', 'Common', 'detail: not found key'), -- common, dr-manager 에서 사용
('not_found_user', 'CDM-Cloud', 'Common', 'detail: not found user \n user ID: <%= id %>'), -- identity, notification 에서 사용
('not_found_group', 'CDM-Cloud', 'Common', 'detail: not found group \n group ID: <%= id %>'), -- identity, center-manager 에서 사용
('not_found_cluster', 'CDM-Cloud', 'Common', 'detail: not found cluster \n cluster ID: <%= cluster_id %>') -- center-manager, dr-manager 에서 사용
;

-- CDM-Cloud.Identity
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('not_reusable_old_password', 'CDM-Cloud', 'Identity', 'detail: not reusable old password'),
('mismatch_password', 'CDM-Cloud', 'Identity', 'detail: current password mismatch'),
('undeletable_user', 'CDM-Cloud', 'Identity', 'detail: undeletable user \n user ID: <%= user_id %>'),
('already_deleted', 'CDM-Cloud', 'Identity', 'detail: already deleted group \n group ID: <%= group_id %>'),
('undeletable_group', 'CDM-Cloud', 'Identity', 'detail: undeletable group \n group ID: <%= group_id %>'),
('already_login', 'CDM-Cloud', 'Identity', 'detail: already login account \n user ID: <%= user_id %>'),
('login_restricted', 'CDM-Cloud', 'Identity', 'detail: account restrict \n account: <%= account %> \n failed count: <%= failed_count %> \n last failed at: <%= last_failed_at %> \n login restricted time: <%= until %>'),
('incorrect_password', 'CDM-Cloud', 'Identity', 'detail: password mismatch'),
('unknown_session', 'CDM-Cloud', 'Identity', 'detail: not found session \n ID: <%= id %>'),
('expired_session', 'CDM-Cloud', 'Identity', 'detail: expired session \n session: <%= session %>'),
('unverified_session', 'CDM-Cloud', 'Identity', 'detail: unverified session \n session: <%= session %> \n cause: <%= cause %>'),
('invalid_session', 'CDM-Cloud', 'Identity', 'detail: invalid session \n session: <%= session %>'),
('not_found_tenant_config', 'CDM-Cloud', 'Identity', 'detail: not found tenant config \n key: <%= key %>'),
('invalid_tenant_config', 'CDM-Cloud', 'Identity', 'detail: invalid tenant config \n key: <%= key %> \n value: <%= value %>'),
('unassignable_role', 'CDM-Cloud', 'Identity', 'detail: unassignable role \n role: <%= role %>')
;

-- CDM-Cloud.Scheduler
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('not_found_schedule', 'CDM-Cloud', 'Scheduler', 'detail: not found schedule \n schedule ID: <%= id %>'),
('invalid_schedule_id', 'CDM-Cloud', 'Scheduler', 'detail: invalid schedule id \n : <%=  %>'),
('invalid_timezone', 'CDM-Cloud', 'Scheduler', 'detail: invalid timezone \n timezone: <%= timezone %>'),
('invalid_topic', 'CDM-Cloud', 'Scheduler', 'detail: invalid topic'),
('invalid_message', 'CDM-Cloud', 'Scheduler', 'detail: invalid message'),
('invalid_start_at', 'CDM-Cloud', 'Scheduler', 'detail: invalid start time \n start at: <%= start_at %>'),
('invalid_end_at', 'CDM-Cloud', 'Scheduler', 'detail: invalid end time \n end at: <%= end_at %>'),
('unsupported_timezone', 'CDM-Cloud', 'Scheduler', 'detail: unsupported schedule type \n type: <%= type %>'),
('unsupported_schedule_type', 'CDM-Cloud', 'Scheduler', 'detail: unsupported timezone \n : <%= timezone %>')
;

-- CDM-Cloud.Monitor
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('not_found_config', 'CDM-Cloud', 'Monitor', 'detail: not found Config \n key: <%= key %>'),
('not_found_node', 'CDM-Cloud', 'Monitor', 'detail: not found node \n node ID: <%= id %>'),
('not_found_service', 'CDM-Cloud', 'Monitor', 'detail: not found telemetry service \n service name: <%= name %>'),
('connection_failed', 'CDM-Cloud', 'Monitor', 'detail: could not connect to telemetry service'),
('no_result', 'CDM-Cloud', 'Monitor', 'detail: could not get result \n query: <%= query %>')
;

-- CDM-Cloud.License
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('not_found_license', 'CDM-Cloud', 'License', 'detail: not found license'),
('decrypt_fail', 'CDM-Cloud', 'License', 'detail: decrypt failed \n cause: <%= cause %>')
;

-- CDM-Cloud.Notification
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('unknown_user', 'CDM-Cloud', 'Notification', 'detail: no role of solution \n solution: <%= solution %> \n user ID: <%= user_id %>'),
('not_found_event', 'CDM-Cloud', 'Notification', 'detail: not found event \n event ID: <%= id %>')
;

-- CDM-Center.Manager
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('not_found_volume_type', 'CDM-Center', 'Manager', 'detail: not found volume type \n volume name: <%= volume_name %> \n volume type name: <%= volume_type_name %>'),
('not_found_volume_backend_name', 'CDM-Center', 'Manager', 'detail: not found volume backend name \n volume type name: <%= volume_type_name %>'),
('not_found_available_service_host_name', 'CDM-Center', 'Manager', 'detail: not found available service host name \n backend name: <%= backend_name %>'),
('not_found_volume_source_name', 'CDM-Center', 'Manager', 'detail: not found volume source name'),
('not_found_snapshot_source_name', 'CDM-Center', 'Manager', 'detail: not found snapshot source name'),
('not_found_instance_hypervisor', 'CDM-Center', 'Manager', 'detail: not found hypervisor of instance \n instance name: <%= instance_name %>'),
('not_found_instance_spec', 'CDM-Center', 'Manager', 'detail: not found instance spec \n instance name: <%= instance_name %> \n flavor name: <%= flavor_name %>'),
('unknown_instance_state', 'CDM-Center', 'Manager', 'detail: unknown instance state \n instance name: <%= instance_name %> \n state: <%= state %>'),
('import_volume_rollback_failed', 'CDM-Center', 'Manager', 'detail: fail to rollback import volume \n volume pair: <%= volume_pair %> \n snapshot pair list: <%= snapshot_pair_list %>'),
('not_found_NFS_export_path', 'CDM-Center', 'Manager', 'detail: not found NFS export path \n volume UUID: <%= volume_uuid %>'),
('unusable_mirror_volume', 'CDM-Center', 'Manager', 'detail: unusable mirror volume \n mirror volume: <%= mirror_volume %> \n cause: <%= cause %>'),
('not_found_managed_volume', 'CDM-Center', 'Manager', 'detail: not found managed volume \n volume pair: <%= volume_pair %>'),
('not_found', 'CDM-Center', 'Manager', 'detail: not found request resource \n cause: <%= cause %>'),
('unauthenticated', 'CDM-Center', 'Manager', 'detail: unauthenticated user \n cause: <%= cause %>'),
('bad_request', 'CDM-Center', 'Manager', 'detail: bad request \n cause: <%= cause %>'),
('unauthorized', 'CDM-Center', 'Manager', 'detail: unauthorized user \n cause: <%= cause %>'),
('unauthorized_user', 'CDM-Center', 'Manager', 'detail: unauthorized user \n user name: <%= user_name %> \n required role name: <%= required_role_name %>'),
('not_connected', 'CDM-Center', 'Manager', 'detail: not connected to cluster \n type code: <%= type_code %>'),
('conflict', 'CDM-Center', 'Manager', 'detail: conflict error \n cause: <%= cause %>'),
('not_found_endpoint', 'CDM-Center', 'Manager', 'detail: not found endpoint \n service type: <%= service_type %>'),
('remote_server_error', 'CDM-Center', 'Manager', 'detail: cluster server error \n cause: <%= cause %>'),
('not_found_cluster_hypervisor', 'CDM-Center', 'Manager', 'detail: not found cluster hypervisor \n cluster hypervisor ID: <%= cluster_hypervisor_id %>'),
('not_found_cluster_network', 'CDM-Center', 'Manager', 'detail: not found cluster network \n cluster network ID: <%= cluster_network_id %>'),
('not_found_cluster_network_by_uuid', 'CDM-Center', 'Manager', 'detail: not found cluster network by uuid \n cluster network UUID: <%= cluster_network_uuid %>'),
('not_found_cluster_subnet', 'CDM-Center', 'Manager', 'detail: not found cluster subnet \n cluster subnet ID: <%= cluster_subnet_id %>'),
('not_found_cluster_floating_ip', 'CDM-Center', 'Manager', 'detail: not found cluster floating ip \n cluster floating IP ID: <%= cluster_floating_ip_id %>'),
('not_found_cluster_tenant', 'CDM-Center', 'Manager', 'detail: not found cluster tenant \n cluster tenant ID: <%= cluster_tenant_id %>'),
('not_found_cluster_tenant_by_uuid', 'CDM-Center', 'Manager', 'detail: not found cluster tenant by uuid \n cluster tenant UUID: <%= cluster_tenant_uuid %>'),
('not_found_cluster_instance', 'CDM-Center', 'Manager', 'detail: not found cluster instance \n cluster instance ID: <%= cluster_instance_id %>'),
('not_found_cluster_instance_by_uuid', 'CDM-Center', 'Manager', 'detail: not found cluster instance by uuid \n cluster instance UUID: <%= cluster_instance_uuid %>'),
('not_found_cluster_instance_spec', 'CDM-Center', 'Manager', 'detail: not found cluster instance spec \n cluster instance spec ID: <%= cluster_instance_spec_id %>'),
('not_found_cluster_instance_spec_by_uuid', 'CDM-Center', 'Manager', 'detail: not found cluster instance spec by uuid \n cluster instance spec UUID: <%= cluster_instance_spec_uuid %>'),
('not_found_cluster_keypair', 'CDM-Center', 'Manager', 'detail: not found cluster keypair \n cluster keypair ID: <%= cluster_keypair_id %>'),
('unsupported_cluster_type', 'CDM-Center', 'Manager', 'detail: unsupported cluster type \n type code: <%= type_code %>'),
('not_found_cluster_availability_zone', 'CDM-Center', 'Manager', 'detail: not found cluster availability zone \n cluster availability zone ID: <%= cluster_availability_zone_id %>'),
('not_found_cluster_volume', 'CDM-Center', 'Manager', 'detail: not found cluster volume \n cluster volume ID: <%= cluster_volume_id %>'),
('not_found_cluster_volume_by_uuid', 'CDM-Center', 'Manager', 'detail: not found cluster volume by uuid \n cluster volume UUID: <%= cluster_volume_uuid %>'),
('not_found_cluster_storage', 'CDM-Center', 'Manager', 'detail: not found cluster storage \n cluster storage ID: <%= cluster_storage_id %>'),
('not_found_router_external_network', 'CDM-Center', 'Manager', 'detail: not found relational external network \n external network ID: <%= cluster_network_id %>'),
('not_found_cluster_router', 'CDM-Center', 'Manager', 'detail: not found cluster router \n cluster router ID : <%= cluster_router_id %>'),
('not_found_cluster_router_by_uuid', 'CDM-Center', 'Manager', 'detail: not found cluster router by uuid \n cluster router UUID: <%= cluster_router_uuid %>'),
('not_found_cluster_security_group', 'CDM-Center', 'Manager', 'detail: not found cluster security group \n cluster security group ID: <%= cluster_security_group_id %>'),
('not_found_cluster_security_group_by_uuid', 'CDM-Center', 'Manager', 'detail: not found cluster security group by uuid \n cluster security group UUID: <%= cluster_security_group_uuid %>'),
('not_found_cluster_sync_status', 'CDM-Center', 'Manager', 'detail: not found cluster sync status \n cluster ID: <%= cluster_id %>'),
('current_password_mismatch', 'CDM-Center', 'Manager', 'detail: current password mismatch \n cluster ID: <%= cluster_id %>'),
('not_found_storage', 'CDM-Center', 'Manager', 'detail: not found storage \n cluster ID: <%= cluster_id %> \n cluster storage ID: <%= cluster_storage_id %>'),
('unsupported_storage_type', 'CDM-Center', 'Manager', 'detail: unsupported storage type \n storage type: <%= storage_type %>'),
('not_found_storage_metadata', 'CDM-Center', 'Manager', 'detail: not found storage metadata \n cluster ID: <%= cluster_id %> \n cluster storage ID: <%= cluster_storage_id %>'),
('volume_status_timeout', 'CDM-Center', 'Manager', 'detail: volume status timeout \n status: <%= status %> \n volume UUID: <%= volume_uuid %>'),
('instance_status_timeout', 'CDM-Center', 'Manager', 'detail: instance status timeout \n status: <%= status %> \n instance UUID: <%= instance_uuid %>'),
('not_found_volume', 'CDM-Center', 'Manager', 'detail: not found volume \n cluster ID: <%= cluster_id %> \n cluster volume ID: <%= cluster_volume_id %>'),
('not_found_volume_metadata', 'CDM-Center', 'Manager', 'detail: not found volume metadata \n cluster ID: <%= cluster_id %> \n cluster storage ID: <%= cluster_storage_id %>'),
('unable_to_synchronize_because_of_status', 'CDM-Center', 'Manager', 'detail: unable to synchronize because of status \n cluster ID: <%= cluster_id %> \n status: <%= status %>')
;

-- CDM-DisasterRecovery.Common
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('not_found_mirror_environment', 'CDM-DisasterRecovery', 'Common', 'detail: not found mirror environment \n source storage ID: <%= source_storage_id %> \n target storage ID: <%= target_storage_id %>'),
('not_found_mirror_environment_status', 'CDM-DisasterRecovery', 'Common', 'detail: not found mirror environment status \n source storage ID: <%= source_storage_id %> \n target storage ID: <%= target_storage_id %>'),
('not_found_mirror_environment_operation', 'CDM-DisasterRecovery', 'Common', 'detail: not found mirror environment operation \n source storage ID: <%= source_storage_id %> \n target storage ID: <%= target_storage_id %>'),
('unknown_mirror_environment_operation', 'CDM-DisasterRecovery', 'Common', 'detail: invalid mirror environment operation value \n operation: <%= operation %>'),
('unknown_mirror_environment_state', 'CDM-DisasterRecovery', 'Common', 'detail: invalid mirror environment state value \n state: <%= state %>'),
('not_found_mirror_volume', 'CDM-DisasterRecovery', 'Common', 'detail: not found mirror volume \n source storage ID: <%= source_storage_id %> \n target storage ID: <%= target_storage_id %> \n source volume ID: <%= source_volume_id %>'),
('not_found_mirror_volume_operation', 'CDM-DisasterRecovery', 'Common', 'detail: not found mirror volume operation \n source storage ID: <%= source_storage_id %> \n target storage ID: <%= target_storage_id %> \n source volume ID: <%= source_volume_id %>'),
('not_found_mirror_volume_status', 'CDM-DisasterRecovery', 'Common', 'detail: not found mirror volume status \n source storage ID: <%= source_storage_id %> \n target storage ID: <%= target_storage_id %> \n source volume ID: <%= source_volume_id %>'),
('unknown_mirror_volume_operation', 'CDM-DisasterRecovery', 'Common', 'detail: invalid mirror volume operation value \n operation: <%= operation %>'),
('unknown_mirror_volume_state', 'CDM-DisasterRecovery', 'Common', 'detail: invalid mirror volume state value \n state: <%= state %>'),
('not_found_mirror_volume_target_metadata', 'CDM-DisasterRecovery', 'Common', 'detail: not found mirror volume target metadata \n source storage ID: <%= source_storage_id %> \n target storage ID: <%= target_storage_id %> \n source volume ID: <%= source_volume_id %>'),
('not_found_mirror_volume_target_agent', 'CDM-DisasterRecovery', 'Common', 'detail: not found mirror volume target agent \n source storage ID: <%= source_storage_id %> \n target storage ID: <%= target_storage_id %> \n source volume ID: <%= source_volume_id %>'),
('volume_existed', 'CDM-DisasterRecovery', 'Common', 'detail: mirror volume exists'),
('not_found_source_volume_reference', 'CDM-DisasterRecovery', 'Common', 'detail: not found source volume reference \n source storage ID: <%= source_storage_id %> \n source volume ID: <%= source_volume_id %>'),
('unknown_job_state_code', 'CDM-DisasterRecovery', 'Common', 'detail: invalid job state \n state code: <%= state_code %>'),
('unknown_job_result_code', 'CDM-DisasterRecovery', 'Common', 'detail: invalid job result \n result code: <%= result_code %>'),
('unknown_job_operation', 'CDM-DisasterRecovery', 'Common', 'detail: invalid job operation \n operation: <%= operation %>'),
('not_found_job', 'CDM-DisasterRecovery', 'Common', 'detail: not found job \n job ID: <%= job_id %>'),
('not_found_job_result', 'CDM-DisasterRecovery', 'Common', 'detail: not found job result \n job ID: <%= job_id %>'),
('not_found_job_instance_status', 'CDM-DisasterRecovery', 'Common', 'detail: not found job instance result \n job ID: <%= job_id %> \n instance ID: <%= instance_id %>'),
('not_found_job_volume_status', 'CDM-DisasterRecovery', 'Common', 'detail: not found job volume result \n job ID: <%= job_id %> \n volume ID: <%= volume_id %>'),
('not_found_job_log', 'CDM-DisasterRecovery', 'Common', 'detail: not found job log \n job ID: <%= job_id %> \n log sequence number: <%= log_seq %>'),
('not_found_task', 'CDM-DisasterRecovery', 'Common', 'detail: not found task \n job ID: <%= job_id %> \n task ID: <%= task_id %>'),
('not_found_clear_task', 'CDM-DisasterRecovery', 'Common', 'detail: not found clear task \n job ID: <%= job_id %> \n task ID: <%= task_id %>'),
('not_found_task_result', 'CDM-DisasterRecovery', 'Common', 'detail: not found task result \n job ID: <%= job_id %> \n task ID: <%= task_id %>'),
('not_shared_task_type', 'CDM-DisasterRecovery', 'Common', 'detail: not shared task type \n task type: <%= task_type %>'),
('not_found_shared_task', 'CDM-DisasterRecovery', 'Common', 'detail: not found shared task \n shared task key: <%= shared_task_key %>'),
('not_found_job_detail', 'CDM-DisasterRecovery', 'Common', 'detail: not found job detail \n job ID: <%= job_id %>'),
('unavailable_storage_existed', 'CDM-DisasterRecovery', 'Common', 'detail: unavailable storage is existed \n storages: <%= storages %>')
;

-- CDM-DisasterRecovery.Manager
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('server_busy', 'CDM-DisasterRecovery', 'Manager', 'detail: server busy \n cause: <%= cause %>'),
('not_pausable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not pausable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_extendable_pausing_time_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not extendable pause time state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_resumable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not resumable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_cancelable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not cancelable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_rollback_retryable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not rollback retryable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_extendable_rollback_time_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not extendable simulation rollback time state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_rollback_ignorable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not rollback ignorable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_rollbackable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not rollbackable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_confirmable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not confirmable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_confirm_retryable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not confirm retryable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_confirm_cancelable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not confirm cancelable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_retryable_state', 'CDM-DisasterRecovery', 'Manager', 'detail: not retryable state \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('ipc_bad_request', 'CDM-DisasterRecovery', 'Manager', 'detail: ipc failed bad request \n code: <%= code %> \n message: <%= message %>'),
('ipc_no_content', 'CDM-DisasterRecovery', 'Manager', 'detail: ipc no content'),
('inactive_cluster', 'CDM-DisasterRecovery', 'Manager', 'detail: inactive cluster \n protection group ID: <%= protection_group_id %> \n cluster ID: <%= cluster_id %>'),
('not_found_instance', 'CDM-DisasterRecovery', 'Manager', 'detail: not found cluster instance \n cluster ID: <%= cluster_id %>'),
('not_found_external_routing_interface', 'CDM-DisasterRecovery', 'Manager', 'detail: not found external network routing interface \n router ID: <%= router_id %>'),
('different_instances_list', 'CDM-DisasterRecovery', 'Manager', 'detail: different instances list of protection group and plan detail \n protection group ID: <%= protection_group_id %> \n plan ID: <%= plan_id %>'),
('unavailable_instance_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: unavailable instance is existed \n instances: <%= instances %>'),
('insufficient_storage_space', 'CDM-DisasterRecovery', 'Manager', 'detail: insufficient storage space \n protection group ID: <%= protection_group_id %> \n plan ID: <%= plan_id %> \n required size: <%= required_size %> \n free size: <%= free_size %>'),
('not_found_protection_group', 'CDM-DisasterRecovery', 'Manager', 'detail: not found protection group \n protection group ID: <%= protection_group_id %> \n tenant ID: <%= tenant_id %>'),
('protection_group_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: protection group is existed \n cluster ID: <%= cluster_id %>'),
('not_found_consistency_group', 'CDM-DisasterRecovery', 'Manager', 'detail: not found consistency group \n protection group ID: <%= protection_group_id %> \n cluster storage ID: <%= cluster_storage_id %> \n cluster tenant ID: <%= cluster_tenant_id %>'),
('not_found_recovery_plan', 'CDM-DisasterRecovery', 'Manager', 'detail: not found recovery plan \n protection group ID: <%= protection_group_id %> \n recovery plan ID: <%= recovery_plan_id %>'),
('not_found_recovery_plans', 'CDM-DisasterRecovery', 'Manager', 'detail: not found recovery plans \n protection group ID: <%= protection_group_id %>'),
('recovery_plan_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: recovery plan is existed \n cluster ID: <%= cluster_id %> \n protection_group ID: <%= protection_group_id %>'),
('failback_recovery_plan_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: failback recovery plan existed \n protection group ID: <%= protection_group_id %> \n recovery plan ID: <%= recovery_plan_id %>'),
('not_found_tenant_recovery_plan', 'CDM-DisasterRecovery', 'Manager', 'detail: not found tenant recovery plan \n recovery plan ID: <%= recovery_plan_id %> \n tenant ID: <%= tenant_id %>'),
('not_found_availability_zone_recovery_plan', 'CDM-DisasterRecovery', 'Manager', 'detail: not found availability zone recovery plan \n recovery plan ID: <%= recovery_plan_id %> \n availability zone ID: <%= availability_zone_id %>'),
('not_found_external_network_recovery_plan', 'CDM-DisasterRecovery', 'Manager', 'detail: not found external network recovery plan \n recovery plan ID: <%= recovery_plan_id %> \n external network ID: <%= external_network_id %>'),
('not_found_floating_ip_recovery_plan', 'CDM-DisasterRecovery', 'Manager', 'detail: not found floating ip recovery plan \n recovery plan ID: <%= recovery_plan_id %> \n floating ip ID: <%= floating_ip_id %>'),
('not_found_storage_recovery_plan', 'CDM-DisasterRecovery', 'Manager', 'detail: not found storage recovery plan \n recovery plan ID: <%= recovery_plan_id %> \n storage ID: <%= storage_id %>'),
('not_found_volume_recovery_plan', 'CDM-DisasterRecovery', 'Manager', 'detail: not found volume recovery plan \n recovery plan ID: <%= recovery_plan_id %> \n volume ID: <%=volume_id  %>'),
('not_found_instance_recovery_plan', 'CDM-DisasterRecovery', 'Manager', 'detail: not found instance recovery plan \n recovery plan ID: <%= recovery_plan_id %> \n instance ID: <%= instance_id %>'),
('floating_ip_address_duplicated', 'CDM-DisasterRecovery', 'Manager', 'detail: protection instance floating IP address is duplicated of recovery cluster floating IP address \n protection instance UUID: <%= protection_instance_uuid %> \n floating IP address: <%= floating_ip_address %>'),
('not_found_recovery_security_group', 'CDM-DisasterRecovery', 'Manager', 'detail: not found recovery security group \n protection group ID: <%= protection_group_id %> \n cluster security group ID: <%= cluster_security_group_id %>'),
('not_found_recovery_job', 'CDM-DisasterRecovery', 'Manager', 'detail: not found recovery job \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('recovery_job_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: recovery job is existed \n protection group ID: <%= protection_group_id %> \n recovery plan ID: <%= recovery_plan_id %>'),
('recovery_job_running', 'CDM-DisasterRecovery', 'Manager', 'detail: recovery job state code is running \n protection group ID: <%= protection_group_id %> \n recovery plan ID: <%= recovery_plan_id %> \n recovery job ID: <%= recovery_job_id %>'),
('unchangeable_recovery_job', 'CDM-DisasterRecovery', 'Manager', 'detail: unchangeable recovery job \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %>'),
('not_runnable_recovery_job', 'CDM-DisasterRecovery', 'Manager', 'detail: not runnable recovery job \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %> \n cause: <%= cause %>'),
('undeletable_recovery_job', 'CDM-DisasterRecovery', 'Manager', 'detail: undeletable recovery job \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %> \n recovery job state code: <%= recovery_job_state_code %>'),
('already_migration_job_registered', 'CDM-DisasterRecovery', 'Manager', 'detail: already recovery job registered \n protection group ID: <%= protection_group_id %>'),
('not_simulation_recovery_job', 'CDM-DisasterRecovery', 'Manager', 'detail: mismatch job type code \n protection group ID: <%= protection_group_id %> \n recovery job ID: <%= recovery_job_id %> \n recovery job type code: <%= recovery_job_type_code %>'),
('unexpected_job_operation', 'CDM-DisasterRecovery', 'Manager', 'detail: unexpected job operation \n recovery job ID: <%= recovery_job_id %> \n recovery job state: <%= recovery_job_state %> \n recovery job operation: <%= recovery_job_operation %>'),
('not_found_recovery_result', 'CDM-DisasterRecovery', 'Manager', 'detail: not found recovery result \n recovery result ID: <%= recovery_result_id %>'),
('undeletable_recovery_result', 'CDM-DisasterRecovery', 'Manager', 'detail: undeletable recovery result \n recovery result ID: <%= recovery_result_id %> \n recovery_type_code: <%= recovery_type_code %>'),
('protection_group_snapshot_precondition_failed', 'CDM-DisasterRecovery', 'Manager', 'detail: precondition failed to create protection group snapshot \n protection group ID: <%= protection_group_id %> \n protection group snapshot ID: <%= protection_group_snapshot_id %> \n cause: <%= cause %>'),
('snapshot_creation_timeout', 'CDM-DisasterRecovery', 'Manager', 'detail: snapshot creation timeout \n protection group ID: <%= protection_group_id %>'),
('snapshot_creation_time_has_passed', 'CDM-DisasterRecovery', 'Manager', 'detail: snapshot creation time has passed \n protection group ID: <%= protection_group_id %>'),
('not_found_recovery_point_snapshot', 'CDM-DisasterRecovery', 'Manager', 'detail: not found recovery point snapshot \n protection group snapshot ID: <%= protection_group_snapshot_id %>'),
('not_found_recovery_plan_snapshot', 'CDM-DisasterRecovery', 'Manager', 'detail: not found recovery plan snapshot \n protection group ID: <%= protection_group_id %> \n recovery plan ID: <%= recovery_plan_id %> \n protection group snapshot ID: <%= protection_group_snapshot_id %>'),
('not_found_instance_recovery_plan_snapshot', 'CDM-DisasterRecovery', 'Manager', 'detail: not found instance recovery plan snapshot \n recovery plan ID: <%= recovery_plan_id %> \n snapshot ID: <%= snapshot_id %> \n instance ID: <%= instance_id %>'),
('not_found_volume_recovery_plan_snapshot', 'CDM-DisasterRecovery', 'Manager', 'detail: not found volume recovery plan snapshot \n recovery plan ID: <%= recovery_plan_id %> \n snapshot ID: <%= snapshot_id %> \n volume ID: <%= volume_id %>'),
('not_deletable_snapshot_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: deletable snapshot not existed \n protection group ID: <%= protection_group_id %>'),
('not_existed_creatable_plans', 'CDM-DisasterRecovery', 'Manager', 'detail: not existed creatable plans \n protection group ID: <%= protection_group_id %>'),
('stopping_mirror_environment_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: stopping mirror environment is existed \n source cluster storage ID: <%= source_cluster_storage_id %> \n target cluster storage ID: <%= target_cluster_storage_id %>'),
('stopping_mirror_volume_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: stopping mirror volume is existed \n source cluster storage ID: <%= source_cluster_storage_id %> \n target cluster storage ID: <%= target_cluster_storage_id %> \n cluster volume ID: <%= cluster_volume_id %>'),
('unknown_cluster_node', 'CDM-DisasterRecovery', 'Manager', 'detail: unknown cluster node'),
('expired_license', 'CDM-DisasterRecovery', 'Manager', 'detail: expired license \n now: <%= now %> \n expire date: <%= expire_date %>'),
('instance_number_exceeded', 'CDM-DisasterRecovery', 'Manager', 'detail: maximum number of instances allowed exceeded \n added or updated: <%=added_or_updated %> \n registered: <%= registered %> \n limit: <%= limit %>'),
('abnormal_state', 'CDM-DisasterRecovery', 'Manager', 'detail: abnormal state'),
('not_existed_replicator_snapshot_file', 'CDM-DisasterRecovery', 'Manager', 'detail: not exist replicator snapshot file'),
('storage_keyring_registration_required', 'CDM-DisasterRecovery', 'Manager', 'detail: storage keyring registration required \n cluster ID: <%= cluster_id %> \n storage UUID: <%= storage_uuid %>'),
('volume_is_already_grouped', 'CDM-DisasterRecovery', 'Manager', 'detail: volume is already grouped \n cluster ID: <%= cluster_id %> \n instance ID: <%= instance_id %> \n volume uuid: <%= volume_uuid %>'),
('insufficient_recovery_hypervisor_resource', 'CDM-DisasterRecovery', 'Manager', 'detail: insufficient recovery hypervisor resource \n recovery cluster ID: <%= recovery_cluster_id %> \n insufficient hypervisor info: <%= insufficient_hypervisor_info %>'),
('duplicated_recovery_job_next_run_time', 'CDM-DisasterRecovery', 'Manager', 'detail: recovery cluster job next run time is duplicated'),
('add_snapshot_failed', 'CDM-DisasterRecovery', 'Manager', 'detail: add snapshot failed'),
('parent_volume_is_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: parent volume is existed \n cluster ID: <%= cluster_id %> \n volume UUID: <%= volume_uuid %>'),
('same_name_is_already_existed', 'CDM-DisasterRecovery', 'Manager', 'detail: same name is already existed \n cluster ID: <%= cluster_id %> \n name: <%= name %>'),
('not_found_plans_creatable_snapshot', 'CDM-DisasterRecovery', 'Manager', 'detail: not found snapshot creatable plan \n protection group ID: <%= protection_group_id %>'),
('volume_is_not_mirroring', 'CDM-DisasterRecovery', 'Manager', 'detail: volume state is not mirroring \n protection cluster volume ID: <%= protection_cluster_volume_id %> \n state code: <%= state_code %>'),
('not_found_owner_group', 'CDM-DisasterRecovery', 'Manager', 'detail: not found owner group \n owner group ID: <%= owner_group_id %>'),
('mismatch_owner_group_id', 'CDM-DisasterRecovery', 'Manager', 'detail: mismatch owner group \n owner group ID ID: <%= owner_group_id %>')
;

-- CDM-DisasterRecovery.Snapshot
INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
('not_found_protection_group_snapshot', 'CDM-DisasterRecovery', 'Snapshot', 'detail: not found protection group snapshot \n protection group ID: <%= protection_group_id %> \n protection group snapshot ID: <%= protection_group_snapshot_id %>'),
('not_found_plan_snapshot', 'CDM-DisasterRecovery', 'Snapshot', 'detail: not found plan snapshot \n protection group ID: <%= protection_group_id %> \n protection group snapshot ID: <%= protection_group_snapshot_id %> \n plan ID: <%= plan_id %>'),
('all_plan_snapshot_creation_failed', 'CDM-DisasterRecovery', 'Snapshot', 'detail: all plan snapshot creation failed \n protection group ID: <%= protection_group_id %> \n protection group snapshot ID: <%= protection_group_snapshot_id %>'),
('not_volume_mirroring_state', 'CDM-DisasterRecovery', 'Snapshot', 'detail: not volume mirroring state \n mirror volume: <%= mirror_volume %>'),
('not_found_mirror_volume_snapshot', 'CDM-DisasterRecovery', 'Snapshot', 'detail: not found mirror volume snapshot \n mirror volume: <%= mirror_volume %> \n snapshot UUID: <%= snapshot_uuid %>'),
('not_supported_storage_type', 'CDM-DisasterRecovery', 'Snapshot', 'detail: not supported storage type \n storage type: <%= storage_type %>')
;