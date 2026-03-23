-- Create assistant_prompt_stages table for multi-prompt support
CREATE TABLE IF NOT EXISTS assistant_prompt_stages (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT NOT NULL,
    updated_by BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    assistant_provider_model_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    template JSONB,
    is_default BOOLEAN DEFAULT false,
    order_idx INT DEFAULT 0,
    transition_rules TEXT
);

CREATE INDEX idx_prompt_stages_provider_model ON assistant_prompt_stages(assistant_provider_model_id);
CREATE INDEX idx_prompt_stages_is_default ON assistant_prompt_stages(is_default);

-- Add current_stage_id to assistant_conversations
ALTER TABLE assistant_conversations ADD COLUMN IF NOT EXISTS current_stage_id BIGINT DEFAULT 0;