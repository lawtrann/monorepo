-- Database initialization script for local development
-- Creates schemas and roles for all services

CREATE SCHEMA IF NOT EXISTS casdoor;
CREATE SCHEMA IF NOT EXISTS mastermgmt;
CREATE SCHEMA IF NOT EXISTS eureka;

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'casdoor_user') THEN
    CREATE ROLE casdoor_user LOGIN PASSWORD 'casdoor_pass';
  END IF;
END
$$;
ALTER ROLE casdoor_user SET search_path TO casdoor;
GRANT ALL ON SCHEMA casdoor TO casdoor_user;
