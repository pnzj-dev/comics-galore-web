-- Comments
CREATE TRIGGER trg_set_comment_depth
    BEFORE INSERT
    ON comments
    FOR EACH ROW
EXECUTE PROCEDURE set_comment_depth();

-- Trigger Function: Updates conversation timestamp AND last_message_id
CREATE OR REPLACE FUNCTION func_on_new_message()
    RETURNS TRIGGER AS
$$
BEGIN
    UPDATE conversations
    SET updated_at      = NOW(),
        last_message_id = NEW.id
    WHERE id = NEW.conversation_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_on_new_message
    AFTER INSERT
    ON messages
    FOR EACH ROW
EXECUTE FUNCTION func_on_new_message();

