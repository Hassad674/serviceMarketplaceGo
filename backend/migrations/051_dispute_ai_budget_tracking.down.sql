ALTER TABLE disputes
    DROP COLUMN IF EXISTS ai_summary_input_tokens,
    DROP COLUMN IF EXISTS ai_summary_output_tokens,
    DROP COLUMN IF EXISTS ai_chat_input_tokens,
    DROP COLUMN IF EXISTS ai_chat_output_tokens,
    DROP COLUMN IF EXISTS ai_budget_bonus_tokens;
