-- =============================================
-- Comment Depth Management
-- =============================================
CREATE OR REPLACE FUNCTION set_comment_depth() RETURNS TRIGGER AS
$$
BEGIN
    -- If there is no parent, it's a top-level comment (depth 0)
    IF NEW.parent_id IS NULL THEN
        NEW.depth := 0;
    ELSE
        -- Fetch parent depth and increment. parent_id is now a UUID.
        SELECT (depth + 1)
        INTO NEW.depth
        FROM comments
        WHERE id = NEW.parent_id;

        -- Fallback if parent isn't found (though FK should prevent this)
        IF NEW.depth IS NULL THEN NEW.depth := 0; END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================
-- Messaging Helper (Atomic Direct Conversation)
-- =============================================
CREATE OR REPLACE FUNCTION get_or_create_direct_conversation(
    p_user1 TEXT, -- Better-Auth uses TEXT for IDs
    p_user2 TEXT
) RETURNS BIGINT AS
$$
DECLARE
    v_conv_id BIGINT;
BEGIN
    -- 1. Try to find an existing 1-on-1 conversation
    -- We look for a conversation where EXACTLY user1 and user2 are participants
    SELECT cp1.conversation_id
    INTO v_conv_id
    FROM conversation_participants cp1
             JOIN conversation_participants cp2 ON cp1.conversation_id = cp2.conversation_id
             JOIN conversations c ON c.id = cp1.conversation_id
    WHERE c.is_group = FALSE
      AND cp1.user_id = p_user1
      AND cp2.user_id = p_user2
      AND (SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = c.id) = 2
    LIMIT 1;

    IF FOUND THEN
        RETURN v_conv_id;
    END IF;

    -- 2. Create a new conversation if not found
    INSERT INTO conversations (name, is_group)
    VALUES (NULL, FALSE)
    RETURNING id INTO v_conv_id;

    -- 3. Add both participants
    INSERT INTO conversation_participants (conversation_id, user_id)
    VALUES (v_conv_id, p_user1),
           (v_conv_id, p_user2);

    RETURN v_conv_id;

EXCEPTION
    -- Handle race conditions (unique_violation on conversation_participants)
    WHEN OTHERS THEN
        SELECT cp1.conversation_id
        INTO v_conv_id
        FROM conversation_participants cp1
                 JOIN conversation_participants cp2 ON cp1.conversation_id = cp2.conversation_id
        WHERE cp1.user_id = p_user1
          AND cp2.user_id = p_user2
        LIMIT 1;

        RETURN v_conv_id;
END;
$$ LANGUAGE plpgsql;