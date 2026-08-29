-- Key-level model whitelist (api_keys.allowed_models).
--
-- Empty/NULL = allow all models (default, existing keys unchanged). A non-empty
-- list restricts gateway requests to the listed model IDs; entries support a
-- trailing '*' wildcard (e.g. "claude-*"). Validation lives in the service
-- layer (ValidateAPIKeyAllowedModels); the column only stores the cleaned list.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS allowed_models JSONB;
