-- Runs once, on the first start of the postgres container (empty volume).
-- auth_db is created by POSTGRES_DB; every other service database goes here.
CREATE DATABASE file_db;
