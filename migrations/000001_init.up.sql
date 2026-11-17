BEGIN;

-- 普通实体保留软删除后的业务唯一键，关联表解绑采用物理删除。
CREATE TABLE sys_dept (
    id bigint CONSTRAINT pk_dept PRIMARY KEY CONSTRAINT ck_dept_id CHECK (id > 0),
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500),
    parent_id bigint NOT NULL DEFAULT 0 CONSTRAINT ck_dept_parent CHECK (parent_id >= 0 AND parent_id <> id),
    ancestors varchar(500) NOT NULL DEFAULT '0', name varchar(64) NOT NULL,
    sort integer NOT NULL DEFAULT 0, leader varchar(64), phone varchar(20),
    status smallint NOT NULL DEFAULT 1 CONSTRAINT ck_dept_status CHECK (status IN (1, 2))
);
CREATE INDEX idx_dept_parent_id ON sys_dept(parent_id);
CREATE INDEX idx_dept_deleted_at ON sys_dept(deleted_at);
COMMENT ON TABLE sys_dept IS '部门；根父级为 0，祖级路径不含自身';

CREATE TABLE sys_user (
    id bigint CONSTRAINT pk_user PRIMARY KEY CONSTRAINT ck_user_id CHECK (id > 0),
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500),
    username varchar(64) NOT NULL CONSTRAINT uk_user_username UNIQUE
        CONSTRAINT ck_user_username CHECK (username = lower(username) AND length(username) > 0),
    nickname varchar(64) NOT NULL, password varchar(100) NOT NULL,
    email varchar(128), phone varchar(20), avatar varchar(255),
    gender smallint NOT NULL DEFAULT 0 CONSTRAINT ck_user_gender CHECK (gender IN (0, 1, 2)),
    dept_id bigint CONSTRAINT fk_user_dept REFERENCES sys_dept(id) ON DELETE RESTRICT,
    status smallint NOT NULL DEFAULT 1 CONSTRAINT ck_user_status CHECK (status IN (1, 2)),
    login_ip varchar(64), login_at timestamp, pwd_updated_at timestamp
);
CREATE INDEX idx_user_dept_id ON sys_user(dept_id);
CREATE INDEX idx_user_phone ON sys_user(phone);
CREATE INDEX idx_user_deleted_at ON sys_user(deleted_at);
COMMENT ON TABLE sys_user IS '用户；初始及重置密码的 pwd_updated_at 为空';

CREATE TABLE sys_role (
    id bigint CONSTRAINT pk_role PRIMARY KEY CONSTRAINT ck_role_id CHECK (id > 0),
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500), name varchar(64) NOT NULL,
    code varchar(64) NOT NULL CONSTRAINT uk_role_code UNIQUE
        CONSTRAINT ck_role_code CHECK (code = lower(code) AND length(code) > 0),
    sort integer NOT NULL DEFAULT 0,
    data_scope smallint NOT NULL DEFAULT 5 CONSTRAINT ck_role_scope CHECK (data_scope BETWEEN 1 AND 5),
    status smallint NOT NULL DEFAULT 1 CONSTRAINT ck_role_status CHECK (status IN (1, 2))
);
CREATE INDEX idx_role_deleted_at ON sys_role(deleted_at);
COMMENT ON TABLE sys_role IS '角色；数据范围 1全部 2自定义 3本部门 4本部门及以下 5本人';

CREATE TABLE sys_menu (
    id bigint CONSTRAINT pk_menu PRIMARY KEY CONSTRAINT ck_menu_id CHECK (id > 0),
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500),
    parent_id bigint NOT NULL DEFAULT 0 CONSTRAINT ck_menu_parent CHECK (parent_id >= 0 AND parent_id <> id),
    name varchar(64) NOT NULL,
    type smallint NOT NULL CONSTRAINT ck_menu_type CHECK (type IN (1, 2, 3)),
    path varchar(255), component varchar(255), perm varchar(128), icon varchar(64),
    sort integer NOT NULL DEFAULT 0,
    visible smallint NOT NULL DEFAULT 1 CONSTRAINT ck_menu_visible CHECK (visible IN (1, 2)),
    status smallint NOT NULL DEFAULT 1 CONSTRAINT ck_menu_status CHECK (status IN (1, 2))
);
CREATE INDEX idx_menu_parent_id ON sys_menu(parent_id);
CREATE INDEX idx_menu_perm ON sys_menu(perm);
CREATE INDEX idx_menu_deleted_at ON sys_menu(deleted_at);
COMMENT ON TABLE sys_menu IS '目录、菜单与按钮；类型 1目录 2菜单 3按钮';

CREATE TABLE sys_post (
    id bigint CONSTRAINT pk_post PRIMARY KEY CONSTRAINT ck_post_id CHECK (id > 0),
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500),
    code varchar(64) NOT NULL CONSTRAINT uk_post_code UNIQUE
        CONSTRAINT ck_post_code CHECK (code = lower(code) AND length(code) > 0),
    name varchar(64) NOT NULL, sort integer NOT NULL DEFAULT 0,
    status smallint NOT NULL DEFAULT 1 CONSTRAINT ck_post_status CHECK (status IN (1, 2))
);
CREATE INDEX idx_post_deleted_at ON sys_post(deleted_at);
COMMENT ON TABLE sys_post IS '岗位';

CREATE TABLE sys_user_role (
    id bigint CONSTRAINT pk_user_role PRIMARY KEY,
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    user_id bigint NOT NULL CONSTRAINT fk_user_role_user REFERENCES sys_user(id) ON DELETE RESTRICT,
    role_id bigint NOT NULL CONSTRAINT fk_user_role_role REFERENCES sys_role(id) ON DELETE RESTRICT,
    CONSTRAINT uk_user_role UNIQUE(user_id, role_id)
);
CREATE INDEX idx_user_role_role_id ON sys_user_role(role_id);

CREATE TABLE sys_user_post (
    id bigint CONSTRAINT pk_user_post PRIMARY KEY,
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    user_id bigint NOT NULL CONSTRAINT fk_user_post_user REFERENCES sys_user(id) ON DELETE RESTRICT,
    post_id bigint NOT NULL CONSTRAINT fk_user_post_post REFERENCES sys_post(id) ON DELETE RESTRICT,
    CONSTRAINT uk_user_post UNIQUE(user_id, post_id)
);
CREATE INDEX idx_user_post_post_id ON sys_user_post(post_id);

CREATE TABLE sys_role_menu (
    id bigint CONSTRAINT pk_role_menu PRIMARY KEY,
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    role_id bigint NOT NULL CONSTRAINT fk_role_menu_role REFERENCES sys_role(id) ON DELETE RESTRICT,
    menu_id bigint NOT NULL CONSTRAINT fk_role_menu_menu REFERENCES sys_menu(id) ON DELETE RESTRICT,
    CONSTRAINT uk_role_menu UNIQUE(role_id, menu_id)
);
CREATE INDEX idx_role_menu_menu_id ON sys_role_menu(menu_id);

CREATE TABLE sys_role_dept (
    id bigint CONSTRAINT pk_role_dept PRIMARY KEY,
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    role_id bigint NOT NULL CONSTRAINT fk_role_dept_role REFERENCES sys_role(id) ON DELETE RESTRICT,
    dept_id bigint NOT NULL CONSTRAINT fk_role_dept_dept REFERENCES sys_dept(id) ON DELETE RESTRICT,
    CONSTRAINT uk_role_dept UNIQUE(role_id, dept_id)
);
CREATE INDEX idx_role_dept_dept_id ON sys_role_dept(dept_id);

CREATE TABLE sys_dict_type (
    id bigint CONSTRAINT pk_dict_type PRIMARY KEY,
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500), name varchar(64) NOT NULL,
    type varchar(64) NOT NULL CONSTRAINT uk_dict_type_type UNIQUE
        CONSTRAINT ck_dict_type_type CHECK (type = lower(type) AND length(type) > 0),
    status smallint NOT NULL DEFAULT 1 CONSTRAINT ck_dict_type_status CHECK (status IN (1, 2))
);
CREATE INDEX idx_dict_type_deleted_at ON sys_dict_type(deleted_at);

CREATE TABLE sys_dict_data (
    id bigint CONSTRAINT pk_dict_data PRIMARY KEY,
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500),
    dict_type varchar(64) NOT NULL CONSTRAINT fk_dict_data_type REFERENCES sys_dict_type(type) ON DELETE RESTRICT,
    label varchar(64) NOT NULL, value varchar(64) NOT NULL,
    sort integer NOT NULL DEFAULT 0, tag varchar(64), is_default boolean NOT NULL DEFAULT false,
    status smallint NOT NULL DEFAULT 1 CONSTRAINT ck_dict_data_status CHECK (status IN (1, 2)),
    CONSTRAINT uk_dict_data_type_value UNIQUE(dict_type, value)
);
CREATE INDEX idx_dict_data_deleted_at ON sys_dict_data(deleted_at);

CREATE TABLE sys_config (
    id bigint CONSTRAINT pk_config PRIMARY KEY,
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500), name varchar(64) NOT NULL,
    key varchar(128) NOT NULL CONSTRAINT uk_config_key UNIQUE,
    value text NOT NULL, is_system boolean NOT NULL DEFAULT false
);
CREATE INDEX idx_config_deleted_at ON sys_config(deleted_at);
COMMENT ON TABLE sys_config IS '普通业务参数；禁止存储密码、密钥和令牌';

CREATE TABLE sys_file (
    id bigint CONSTRAINT pk_file PRIMARY KEY,
    created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    updated_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    deleted_at timestamp, remark varchar(500),
    name varchar(255) NOT NULL, original_name varchar(255) NOT NULL, path varchar(500) NOT NULL,
    size bigint NOT NULL CONSTRAINT ck_file_size CHECK (size >= 0),
    mime varchar(128), storage varchar(32) NOT NULL DEFAULT 'local', hash varchar(64)
);
CREATE INDEX idx_file_owner_storage_hash ON sys_file(created_by, storage, hash);
CREATE INDEX idx_file_deleted_at ON sys_file(deleted_at);
COMMENT ON TABLE sys_file IS '文件；created_by 为所有者，存储相对路径不直接对外暴露';

-- 日志为只追加记录，不含实体软删除和更新公共字段。
CREATE TABLE sys_oper_log (
    id bigint CONSTRAINT pk_oper_log PRIMARY KEY,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    user_id bigint NOT NULL DEFAULT 0, username varchar(64) NOT NULL DEFAULT '',
    module varchar(64) NOT NULL, action varchar(128) NOT NULL, trace_id varchar(64),
    method varchar(16) NOT NULL, url varchar(500) NOT NULL, ip varchar(64),
    request text, response text,
    status smallint NOT NULL CONSTRAINT ck_oper_log_status CHECK (status IN (1, 2)),
    error_msg text, cost_ms integer NOT NULL CONSTRAINT ck_oper_log_cost CHECK (cost_ms >= 0),
    oper_at timestamp NOT NULL
);
CREATE INDEX idx_oper_log_oper_at ON sys_oper_log(oper_at);
CREATE INDEX idx_oper_log_user_id ON sys_oper_log(user_id);
CREATE INDEX idx_oper_log_trace_id ON sys_oper_log(trace_id);

CREATE TABLE sys_login_log (
    id bigint CONSTRAINT pk_login_log PRIMARY KEY,
    created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
    username varchar(64) NOT NULL, ip varchar(64), location varchar(128),
    browser varchar(64), os varchar(64),
    status smallint NOT NULL CONSTRAINT ck_login_log_status CHECK (status IN (1, 2)),
    msg varchar(255), login_at timestamp NOT NULL
);
CREATE INDEX idx_login_log_username ON sys_login_log(username);
CREATE INDEX idx_login_log_login_at ON sys_login_log(login_at);

COMMIT;
