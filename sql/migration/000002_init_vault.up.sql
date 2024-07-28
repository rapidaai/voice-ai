CREATE TABLE vaults_20240712 AS
SELECT * FROM vaults WHERE 1=0;

-- Step 2: Copy data from the existing table
INSERT INTO vaults_20240712
SELECT * FROM vaults;


CREATE TABLE vaults (
    id BIGINT PRIMARY KEY,
    created_date timestamp without time zone DEFAULT now() NOT NULL,
    updated_date timestamp without time zone,

    vault_type VARCHAR(200) NOT NULL,
    vault_type_id BIGINT NOT NULL,
    
    vault_level VARCHAR(200) NOT NULL,
    vault_level_id BIGINT NOT NULL,

    name VARCHAR(200) NOT NULL,
    value JSON NOT NULL,  -- Assuming gorm_types.InterfaceMap is JSON
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    created_by BIGINT NOT NULL,
    updated_by BIGINT NOT NULL
);

CREATE INDEX idx_vault_type ON vaults(vault_type);
CREATE INDEX idx_vault_type_id ON vaults(vault_type_id);
CREATE INDEX idx_vault_level ON vaults(vault_level);
CREATE INDEX idx_vault_level_id ON vaults(vault_level_id);

CREATE TABLE o_auth_external_connects (
    id BIGINT PRIMARY KEY,
    created_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    identifier VARCHAR(200) NOT NULL,
    tool_connect VARCHAR(200) NOT NULL,
    tool_id BIGINT NOT NULL,
    linker VARCHAR(200) NOT NULL,
    linker_id BIGINT NOT NULL,
    redirect_to VARCHAR(200) NOT NULL
);

CREATE INDEX idx_o_auth_external_connects ON o_auth_external_connects(identifier);
CREATE INDEX linker_id_o_auth_external_connects ON o_auth_external_connects(linker_id);
