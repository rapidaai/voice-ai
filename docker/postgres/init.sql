-- Create databases
CREATE DATABASE assistant_db;
CREATE DATABASE integration_db;
CREATE DATABASE endpoint_db;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE assistant_db TO rapida_user;
GRANT ALL PRIVILEGES ON DATABASE integration_db TO rapida_user;
GRANT ALL PRIVILEGES ON DATABASE endpoint_db TO rapida_user;
