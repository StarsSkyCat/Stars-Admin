-- 任务配置由应用注册函数白名单解释，不执行任意命令。
CREATE TABLE job_task (
 id bigint CONSTRAINT pk_job_task PRIMARY KEY,
 created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
 created_at timestamp NOT NULL, updated_at timestamp NOT NULL, deleted_at timestamp, remark varchar(500),
 name varchar(64) NOT NULL, task_group varchar(64) NOT NULL DEFAULT 'default',
 cron varchar(64) NOT NULL, target varchar(255) NOT NULL, params text,
 misfire_policy smallint NOT NULL DEFAULT 2 CONSTRAINT ck_job_task_misfire CHECK(misfire_policy IN (1,2)),
 concurrent boolean NOT NULL DEFAULT false,
 status smallint NOT NULL DEFAULT 2 CONSTRAINT ck_job_task_status CHECK(status IN (1,2))
);
CREATE INDEX idx_job_task_deleted_at ON job_task(deleted_at);
CREATE TABLE job_log (
 id bigint CONSTRAINT pk_job_log PRIMARY KEY, created_at timestamp NOT NULL,
 task_id bigint NOT NULL, name varchar(64) NOT NULL, task_group varchar(64) NOT NULL,
 target varchar(255) NOT NULL, params text NOT NULL,
 status smallint NOT NULL CONSTRAINT ck_job_log_status CHECK(status IN (1,2)),
 error_msg text NOT NULL DEFAULT '', cost_ms integer NOT NULL CONSTRAINT ck_job_log_cost CHECK(cost_ms>=0),
 started_at timestamp NOT NULL
);
CREATE INDEX idx_job_log_task_id ON job_log(task_id);
CREATE INDEX idx_job_log_started_at ON job_log(started_at);
