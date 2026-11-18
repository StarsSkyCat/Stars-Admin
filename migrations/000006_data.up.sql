BEGIN;

-- 已有业务键（含软删除记录）和现有授权保留原样。固定小 ID 留给基础数据。
DO $$
BEGIN
    IF EXISTS (SELECT FROM sys_role WHERE (id = 1) <> (code = 'admin')) THEN
        RAISE EXCEPTION '保留角色 ID 或编码已被占用';
    END IF;
    IF EXISTS (SELECT FROM sys_user WHERE (id = 1) <> (username = 'admin')) THEN
        RAISE EXCEPTION '保留管理员 ID 或账号已被占用';
    END IF;
END;
$$;

INSERT INTO sys_dept (id, name)
SELECT 1, '总部'
WHERE NOT EXISTS (SELECT FROM sys_dept WHERE parent_id = 0 AND name = '总部');

CREATE TEMP TABLE initial_menu (
    id bigint PRIMARY KEY, parent_id bigint NOT NULL, name text NOT NULL, type smallint NOT NULL,
    path text, legacy_path text, component text, perm text
) ON COMMIT DROP;

INSERT INTO initial_menu (id, parent_id, name, type, path, legacy_path, component, perm) VALUES
    (100, 0, '系统管理', 1, '/system', NULL, NULL, NULL),
    (101, 100, '用户管理', 2, '/system/users', '/system/user', '/system/users/index', NULL),
    (201, 101, '用户管理-list', 3, NULL, NULL, NULL, 'sys:user:list'),
    (202, 101, '用户管理-get', 3, NULL, NULL, NULL, 'sys:user:get'),
    (203, 101, '用户管理-add', 3, NULL, NULL, NULL, 'sys:user:add'),
    (204, 101, '用户管理-edit', 3, NULL, NULL, NULL, 'sys:user:edit'),
    (205, 101, '用户管理-remove', 3, NULL, NULL, NULL, 'sys:user:remove'),
    (206, 101, '用户管理-resetPwd', 3, NULL, NULL, NULL, 'sys:user:resetPwd'),
    (207, 101, '用户管理-grant', 3, NULL, NULL, NULL, 'sys:user:grant'),
    (208, 101, '用户管理-import', 3, NULL, NULL, NULL, 'sys:user:import'),
    (209, 101, '用户管理-export', 3, NULL, NULL, NULL, 'sys:user:export'),
    (102, 100, '角色管理', 2, '/system/role', NULL, NULL, NULL),
    (210, 102, '角色管理-list', 3, NULL, NULL, NULL, 'sys:role:list'),
    (211, 102, '角色管理-get', 3, NULL, NULL, NULL, 'sys:role:get'),
    (212, 102, '角色管理-add', 3, NULL, NULL, NULL, 'sys:role:add'),
    (213, 102, '角色管理-edit', 3, NULL, NULL, NULL, 'sys:role:edit'),
    (214, 102, '角色管理-remove', 3, NULL, NULL, NULL, 'sys:role:remove'),
    (215, 102, '角色管理-grant', 3, NULL, NULL, NULL, 'sys:role:grant'),
    (103, 100, '菜单管理', 2, '/system/menu', NULL, NULL, NULL),
    (216, 103, '菜单管理-list', 3, NULL, NULL, NULL, 'sys:menu:list'),
    (217, 103, '菜单管理-get', 3, NULL, NULL, NULL, 'sys:menu:get'),
    (218, 103, '菜单管理-add', 3, NULL, NULL, NULL, 'sys:menu:add'),
    (219, 103, '菜单管理-edit', 3, NULL, NULL, NULL, 'sys:menu:edit'),
    (220, 103, '菜单管理-remove', 3, NULL, NULL, NULL, 'sys:menu:remove'),
    (104, 100, '部门管理', 2, '/system/departments', '/system/dept', '/system/departments/index', NULL),
    (221, 104, '部门管理-list', 3, NULL, NULL, NULL, 'sys:dept:list'),
    (222, 104, '部门管理-get', 3, NULL, NULL, NULL, 'sys:dept:get'),
    (223, 104, '部门管理-add', 3, NULL, NULL, NULL, 'sys:dept:add'),
    (224, 104, '部门管理-edit', 3, NULL, NULL, NULL, 'sys:dept:edit'),
    (225, 104, '部门管理-remove', 3, NULL, NULL, NULL, 'sys:dept:remove'),
    (105, 100, '岗位管理', 2, '/system/post', NULL, NULL, NULL),
    (226, 105, '岗位管理-list', 3, NULL, NULL, NULL, 'sys:post:list'),
    (227, 105, '岗位管理-get', 3, NULL, NULL, NULL, 'sys:post:get'),
    (228, 105, '岗位管理-add', 3, NULL, NULL, NULL, 'sys:post:add'),
    (229, 105, '岗位管理-edit', 3, NULL, NULL, NULL, 'sys:post:edit'),
    (230, 105, '岗位管理-remove', 3, NULL, NULL, NULL, 'sys:post:remove'),
    (106, 100, '字典管理', 2, '/system/dict', NULL, NULL, NULL),
    (231, 106, '字典管理-list', 3, NULL, NULL, NULL, 'sys:dict:list'),
    (232, 106, '字典管理-get', 3, NULL, NULL, NULL, 'sys:dict:get'),
    (233, 106, '字典管理-add', 3, NULL, NULL, NULL, 'sys:dict:add'),
    (234, 106, '字典管理-edit', 3, NULL, NULL, NULL, 'sys:dict:edit'),
    (235, 106, '字典管理-remove', 3, NULL, NULL, NULL, 'sys:dict:remove'),
    (236, 106, '字典管理-refresh', 3, NULL, NULL, NULL, 'sys:dict:refresh'),
    (107, 100, '参数管理', 2, '/system/config', NULL, NULL, NULL),
    (237, 107, '参数管理-list', 3, NULL, NULL, NULL, 'sys:config:list'),
    (238, 107, '参数管理-get', 3, NULL, NULL, NULL, 'sys:config:get'),
    (239, 107, '参数管理-add', 3, NULL, NULL, NULL, 'sys:config:add'),
    (240, 107, '参数管理-edit', 3, NULL, NULL, NULL, 'sys:config:edit'),
    (241, 107, '参数管理-remove', 3, NULL, NULL, NULL, 'sys:config:remove'),
    (242, 107, '参数管理-refresh', 3, NULL, NULL, NULL, 'sys:config:refresh'),
    (108, 100, '文件管理', 2, '/system/file', NULL, NULL, NULL),
    (243, 108, '文件管理-add', 3, NULL, NULL, NULL, 'sys:file:add'),
    (244, 108, '文件管理-get', 3, NULL, NULL, NULL, 'sys:file:get'),
    (245, 108, '文件管理-remove', 3, NULL, NULL, NULL, 'sys:file:remove'),
    (109, 100, '日志审计', 2, '/system/audit', NULL, NULL, NULL),
    (246, 109, '日志审计-operList', 3, NULL, NULL, NULL, 'sys:audit:operList'),
    (247, 109, '日志审计-operClean', 3, NULL, NULL, NULL, 'sys:audit:operClean'),
    (248, 109, '日志审计-operExport', 3, NULL, NULL, NULL, 'sys:audit:operExport'),
    (249, 109, '日志审计-loginList', 3, NULL, NULL, NULL, 'sys:audit:loginList'),
    (250, 109, '日志审计-loginClean', 3, NULL, NULL, NULL, 'sys:audit:loginClean'),
    (251, 109, '日志审计-loginExport', 3, NULL, NULL, NULL, 'sys:audit:loginExport'),
    (252, 109, '日志审计-onlineList', 3, NULL, NULL, NULL, 'sys:audit:onlineList'),
    (253, 109, '日志审计-forceLogout', 3, NULL, NULL, NULL, 'sys:audit:forceLogout'),
    (110, 0, '定时任务', 2, '/job/task', NULL, NULL, NULL),
    (254, 110, '定时任务-list', 3, NULL, NULL, NULL, 'job:task:list'),
    (255, 110, '定时任务-get', 3, NULL, NULL, NULL, 'job:task:get'),
    (256, 110, '定时任务-add', 3, NULL, NULL, NULL, 'job:task:add'),
    (257, 110, '定时任务-edit', 3, NULL, NULL, NULL, 'job:task:edit'),
    (258, 110, '定时任务-remove', 3, NULL, NULL, NULL, 'job:task:remove'),
    (259, 110, '定时任务-run', 3, NULL, NULL, NULL, 'job:task:run'),
    (111, 0, '任务日志', 2, '/job/log', NULL, NULL, NULL),
    (260, 111, '任务日志-list', 3, NULL, NULL, NULL, 'job:log:list'),
    (261, 111, '任务日志-remove', 3, NULL, NULL, NULL, 'job:log:remove'),
    (112, 0, '代码生成', 2, '/gen/table', NULL, NULL, NULL),
    (262, 112, '代码生成-list', 3, NULL, NULL, NULL, 'gen:table:list'),
    (263, 112, '代码生成-get', 3, NULL, NULL, NULL, 'gen:table:get'),
    (264, 112, '代码生成-import', 3, NULL, NULL, NULL, 'gen:table:import'),
    (265, 112, '代码生成-edit', 3, NULL, NULL, NULL, 'gen:table:edit'),
    (266, 112, '代码生成-remove', 3, NULL, NULL, NULL, 'gen:table:remove'),
    (267, 112, '代码生成-preview', 3, NULL, NULL, NULL, 'gen:table:preview'),
    (268, 112, '代码生成-gen', 3, NULL, NULL, NULL, 'gen:table:gen');

-- 兼容旧种子的雪花 ID 和用户、部门旧路径。子节点沿用实际父级 ID。
CREATE TEMP TABLE initial_menu_resolved ON COMMIT DROP AS
WITH RECURSIVE parents (id, actual_id, actual_parent_id, exists_in_db, deleted) AS (
    SELECT 0::bigint, 0::bigint, 0::bigint, true, false
    UNION ALL
    SELECT seed.id, COALESCE(existing.id, seed.id), parent.actual_id,
           existing.id IS NOT NULL, existing.deleted_at IS NOT NULL
    FROM parents parent
    JOIN initial_menu seed ON seed.parent_id = parent.id
    LEFT JOIN LATERAL (
        SELECT menu.id, menu.deleted_at
        FROM sys_menu menu
        WHERE (seed.perm IS NOT NULL AND menu.perm = seed.perm)
           OR (seed.perm IS NULL AND menu.parent_id = parent.actual_id AND menu.path IN (seed.path, seed.legacy_path))
        ORDER BY menu.id
        LIMIT 1
    ) existing ON true
    WHERE NOT parent.deleted
)
SELECT * FROM parents WHERE id <> 0;

INSERT INTO sys_menu (id, parent_id, name, type, path, component, perm)
SELECT resolved.actual_id, resolved.actual_parent_id, seed.name, seed.type, seed.path, seed.component, seed.perm
FROM initial_menu_resolved resolved
JOIN initial_menu seed USING (id)
WHERE NOT resolved.exists_in_db;

-- 只为本次创建的管理员角色初始化菜单授权，已有角色的授权保持原状。
WITH new_role AS (
    INSERT INTO sys_role (id, name, code, data_scope)
    VALUES (1, '超级管理员', 'admin', 1)
    ON CONFLICT (id) DO NOTHING
    RETURNING id
)
INSERT INTO sys_role_menu (id, role_id, menu_id)
SELECT menu.id, new_role.id, menu.actual_id
FROM initial_menu_resolved menu CROSS JOIN new_role
WHERE NOT menu.deleted;

-- 同一会话可由迁移入口传入自定义 bcrypt 哈希。公开的 123456 哈希只用于开发环境。
-- pwd_updated_at 保持 NULL，管理员首次登录后必须改密。
WITH new_admin AS (
    INSERT INTO sys_user (id, username, nickname, password, dept_id)
    SELECT 1, 'admin', '超级管理员',
        COALESCE(NULLIF(current_setting('stars_admin.initial_password_hash', true), ''),
            CASE WHEN current_setting('stars_admin.development', true) = 'on'
                THEN '$2a$10$ETXqj9MbpTvil8Azff.PDOTVqXdENOb7rtU/4o7EWUpYHlGfsoc6i'
            END),
        (SELECT id FROM sys_dept WHERE parent_id = 0 AND name = '总部' AND deleted_at IS NULL ORDER BY id LIMIT 1)
    WHERE NOT EXISTS (SELECT FROM sys_user WHERE id = 1)
    RETURNING id
)
INSERT INTO sys_user_role (id, user_id, role_id)
SELECT 1, id, 1 FROM new_admin;

INSERT INTO sys_dict_type (id, name, type) VALUES
    (1, '用户性别', 'sys_user_gender'),
    (2, '状态', 'sys_status'),
    (3, '菜单类型', 'sys_menu_type'),
    (4, '数据范围', 'sys_data_scope')
ON CONFLICT (type) DO NOTHING;

INSERT INTO sys_dict_data (id, dict_type, label, value, sort) VALUES
    (1, 'sys_user_gender', '未知', '0', 0),
    (2, 'sys_user_gender', '男', '1', 1),
    (3, 'sys_user_gender', '女', '2', 2),
    (4, 'sys_status', '正常', '1', 0),
    (5, 'sys_status', '停用', '2', 1),
    (6, 'sys_menu_type', '目录', '1', 0),
    (7, 'sys_menu_type', '菜单', '2', 1),
    (8, 'sys_menu_type', '按钮', '3', 2),
    (9, 'sys_data_scope', '全部', '1', 0),
    (10, 'sys_data_scope', '自定义', '2', 1),
    (11, 'sys_data_scope', '本部门', '3', 2),
    (12, 'sys_data_scope', '本部门及以下', '4', 3),
    (13, 'sys_data_scope', '仅本人', '5', 4)
ON CONFLICT (dict_type, value) DO NOTHING;

INSERT INTO sys_config (id, name, key, value, is_system) VALUES
    (1, '验证码开关', 'sys.account.captchaEnabled', 'true', true),
    (2, '多点登录', 'sys.account.multiLogin', 'true', true),
    (3, '会话数量上限', 'sys.account.maxSessions', '5', true),
    (4, '密码重试次数', 'sys.account.passwordRetryLimit', '5', true),
    (5, '账号锁定分钟', 'sys.account.lockMinutes', '10', true)
ON CONFLICT (key) DO NOTHING;

COMMIT;
