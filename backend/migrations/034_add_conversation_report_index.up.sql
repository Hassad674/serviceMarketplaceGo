CREATE INDEX IF NOT EXISTS idx_reports_conversation_status
    ON reports(conversation_id, status)
    WHERE conversation_id IS NOT NULL;
