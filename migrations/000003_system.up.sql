-- 文件去重仅约束同一所有者的有效记录，删除后允许重新上传。
CREATE UNIQUE INDEX uk_file_owner_storage_hash_active ON sys_file(created_by, storage, hash) WHERE deleted_at IS NULL AND hash IS NOT NULL;
-- 持久补偿任务在物理删除成功后移除，应用重启可继续处理。
CREATE TABLE sys_file_cleanup (
 id bigint CONSTRAINT pk_file_cleanup PRIMARY KEY,
 created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
 created_at timestamp NOT NULL, updated_at timestamp NOT NULL,
 path varchar(500) NOT NULL, storage varchar(32) NOT NULL DEFAULT 'local',
 attempts integer NOT NULL DEFAULT 0 CONSTRAINT ck_file_cleanup_attempts CHECK (attempts >= 0),
 next_attempt timestamp NOT NULL
);
CREATE INDEX idx_file_cleanup_next_attempt ON sys_file_cleanup(next_attempt);
