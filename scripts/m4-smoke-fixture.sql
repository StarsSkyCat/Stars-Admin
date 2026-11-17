-- 仅由 smoke 脚本在隔离 stars_admin_smoke 数据库中创建，不是正式迁移。
CREATE TABLE demo_smoke_item (
 id bigint PRIMARY KEY,
 created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
 created_at timestamp NOT NULL, updated_at timestamp NOT NULL,
 deleted_at timestamp, remark varchar(500), name varchar(64) NOT NULL UNIQUE
);
CREATE INDEX idx_demo_smoke_item_deleted_at ON demo_smoke_item(deleted_at);
