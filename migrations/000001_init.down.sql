BEGIN;
-- 仅回退结构，不会恢复删除的数据；生产执行前需完成备份。
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
