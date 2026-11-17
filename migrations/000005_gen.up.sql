CREATE TABLE gen_table (
 id bigint CONSTRAINT pk_gen_table PRIMARY KEY,
 created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
 created_at timestamp NOT NULL, updated_at timestamp NOT NULL, deleted_at timestamp, remark varchar(500),
 table_name varchar(128) NOT NULL CONSTRAINT uk_gen_table_name UNIQUE,
 table_comment varchar(255), module_name varchar(64) NOT NULL,
 feature_name varchar(64) NOT NULL, business_name varchar(64) NOT NULL,
 author varchar(64), gen_type varchar(16) NOT NULL DEFAULT 'zip' CONSTRAINT ck_gen_type CHECK(gen_type IN ('zip','local')),
 options text NOT NULL, synced_at timestamp
);
CREATE INDEX idx_gen_table_deleted_at ON gen_table(deleted_at);
CREATE TABLE gen_table_column (
 id bigint CONSTRAINT pk_gen_column PRIMARY KEY,
 created_by bigint NOT NULL DEFAULT 0, updated_by bigint NOT NULL DEFAULT 0,
 created_at timestamp NOT NULL, updated_at timestamp NOT NULL, deleted_at timestamp, remark varchar(500),
 table_id bigint NOT NULL CONSTRAINT fk_gen_column_table REFERENCES gen_table(id) ON DELETE RESTRICT,
 column_name varchar(128) NOT NULL, column_comment varchar(255), db_type varchar(128) NOT NULL,
 go_type varchar(64) NOT NULL, go_field varchar(64) NOT NULL, json_field varchar(64) NOT NULL,
 is_pk boolean NOT NULL, is_required boolean NOT NULL, is_insert boolean NOT NULL, is_edit boolean NOT NULL,
 is_list boolean NOT NULL, is_query boolean NOT NULL, is_nullable boolean NOT NULL,
 max_length integer, numeric_precision integer, numeric_scale integer, column_default text,
 query_type varchar(16) NOT NULL DEFAULT 'EQ' CONSTRAINT ck_gen_query_type CHECK(query_type IN ('EQ','NE','LIKE','BETWEEN')),
 html_type varchar(16) NOT NULL DEFAULT 'input', dict_type varchar(64), sort integer NOT NULL DEFAULT 0,
 change_kind varchar(16) NOT NULL DEFAULT 'same' CONSTRAINT ck_gen_change CHECK(change_kind IN ('same','added','removed','changed')),
 CONSTRAINT uk_gen_column UNIQUE(table_id,column_name)
);
CREATE INDEX idx_gen_column_table_id ON gen_table_column(table_id);
CREATE INDEX idx_gen_column_deleted_at ON gen_table_column(deleted_at);
