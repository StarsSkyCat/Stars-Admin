BEGIN;
DROP TABLE casbin_rule;
ALTER TABLE sys_user DROP COLUMN auth_version;
COMMIT;
