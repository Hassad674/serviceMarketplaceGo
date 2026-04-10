-- AI budget tracking per dispute. Token counts are cumulative across the
-- dispute lifetime. Summary and chat are tracked separately so the admin
-- UI can show distinct progress bars for each. The bonus column tracks
-- how much extra budget the admin has manually granted via the
-- "Augmenter le budget" button (applied on top of the tier base budget).
ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS ai_summary_input_tokens  INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ai_summary_output_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ai_chat_input_tokens     INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ai_chat_output_tokens    INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ai_budget_bonus_tokens   INTEGER NOT NULL DEFAULT 0;
