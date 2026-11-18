-- 回退基线会删除全部业务表及其数据。

BEGIN;
DROP TABLE IF EXISTS gen_table_column;
DROP TABLE IF EXISTS gen_table;
DROP TABLE IF EXISTS job_log;
DROP TABLE IF EXISTS job_task;
DROP TABLE IF EXISTS sys_file_cleanup;
DROP INDEX IF EXISTS uk_file_owner_storage_hash_active;
DROP TABLE casbin_rule;
ALTER TABLE sys_user DROP COLUMN auth_version;
DROP TABLE sys_login_log;
DROP TABLE sys_oper_log;
DROP TABLE sys_file;
DROP TABLE sys_config;
DROP TABLE sys_dict_data;
DROP TABLE sys_dict_type;
DROP TABLE sys_role_dept;
DROP TABLE sys_role_menu;
DROP TABLE sys_user_post;
DROP TABLE sys_user_role;
DROP TABLE sys_post;
DROP TABLE sys_menu;
DROP TABLE sys_role;
DROP TABLE sys_user;
DROP TABLE sys_dept;

COMMIT;
