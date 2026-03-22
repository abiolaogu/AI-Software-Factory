-- Reverse migration for 00001_init.up.sql
-- Drops tables in dependency order (child first, then parent)

DROP TABLE IF EXISTS work_items;
DROP TABLE IF EXISTS tenants;
