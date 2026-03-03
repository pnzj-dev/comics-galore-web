-- =============================================================================
-- EXTENSIONS & CUSTOM TYPES
-- =============================================================================
CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "pg_uuidv7";
CREATE TYPE user_role AS ENUM ('user', 'admin', 'writer' );
CREATE TYPE membership_plan_type AS ENUM ('free', 'bronze', 'silver', 'gold', 'n/a');
CREATE TYPE reaction_type AS ENUM ('like', 'dislike', 'love', 'laugh', 'angry');
CREATE TYPE target_type_enum AS ENUM ('post', 'comment');

-- =============================================================================
-- CORE ENTITIES
-- =============================================================================

CREATE TABLE users
(
    id              VARCHAR(255) PRIMARY KEY, -- Length based on typical Auth provider IDs
    name            VARCHAR(100)                       NOT NULL,
    email           CITEXT                             NOT NULL,
    email_verified  BOOLEAN              DEFAULT false NOT NULL,
    image           TEXT,                     -- URLs kept as TEXT for length flexibility
    role            user_role            DEFAULT 'user',
    membership_plan membership_plan_type DEFAULT 'free',
    banned          BOOLEAN              DEFAULT false,
    ban_reason      VARCHAR(500),
    ban_expires     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ          DEFAULT now() NOT NULL,
    updated_at      TIMESTAMPTZ          DEFAULT now() NOT NULL,
    CONSTRAINT user_email_unique UNIQUE (email)
);

CREATE TABLE categories
(
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    slug         CITEXT UNIQUE NOT NULL,
    display_name VARCHAR(100)  NOT NULL,
    created_at   TIMESTAMPTZ      DEFAULT NOW()
);

CREATE TABLE blogposts
(
    id            UUID PRIMARY KEY        DEFAULT uuid_generate_v7(),
    title         VARCHAR(255)   NOT NULL,
    author_name   VARCHAR(100)   NOT NULL,
    uploader_id   VARCHAR(255)   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category_id   UUID           NOT NULL REFERENCES categories (id),
    description   TEXT           NOT NULL, -- Keep TEXT for long content (uses TOAST storage)

    -- Optimized Tags: TEXT[] + GIN index is the gold standard for tag filtering
    tags          TEXT[]         NOT NULL DEFAULT '{}',

    -- Structure: {cloudflare_id, backup_s3_key, variants}
    cover         JSONB          NOT NULL,
    -- Structure: [{cloudflare_id, display_order, backup_s3_key, variants}, ...]
    previews      JSONB          NOT NULL CHECK (jsonb_typeof(previews) = 'array' AND
                                                 jsonb_array_length(previews) >= 5 AND
                                                 jsonb_array_length(previews) <= 10),

    rating        NUMERIC(3, 2)  NOT NULL DEFAULT 0.00,
    language_code VARCHAR(10)    NOT NULL DEFAULT 'en',
    pages         INTEGER        NOT NULL DEFAULT 0,
    size_bytes    BIGINT         NOT NULL CHECK (size_bytes >= 0),
    mime_types    VARCHAR(100)[] NOT NULL CHECK (cardinality(mime_types) >= 1),

    created_at    TIMESTAMPTZ             DEFAULT NOW(),
    updated_at    TIMESTAMPTZ             DEFAULT NOW()
);

-- =============================================================================
-- STATS & ARCHIVES
-- =============================================================================

CREATE TABLE blogpost_stats
(
    post_id    UUID PRIMARY KEY REFERENCES blogposts (id) ON DELETE CASCADE,
    comments   BIGINT NOT NULL DEFAULT 0,
    downloads  BIGINT NOT NULL DEFAULT 0,
    auth_views BIGINT NOT NULL DEFAULT 0,
    anon_views BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE archives
(
    id         UUID PRIMARY KEY      DEFAULT uuid_generate_v7(),
    post_id    UUID         NOT NULL REFERENCES blogposts (id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    size_bytes BIGINT       NOT NULL CHECK (size_bytes >= 0),
    pages      INTEGER      NOT NULL DEFAULT 0,
    -- Structure: [{role, s3_key, s3_bucket, endpoint}, ...]
    locations  JSONB        NOT NULL CHECK (jsonb_typeof(locations) = 'array' AND jsonb_array_length(locations) <= 10),
    updated_at TIMESTAMPTZ           DEFAULT NOW(),
    created_at TIMESTAMPTZ           DEFAULT NOW()
);

CREATE TABLE archive_stats
(
    archive_id UUID PRIMARY KEY REFERENCES archives (id) ON DELETE CASCADE,
    downloads  BIGINT NOT NULL DEFAULT 0
);

-- =============================================================================
-- ENGAGEMENT & MESSAGING
-- =============================================================================

CREATE TABLE comments
(
    id         UUID PRIMARY KEY      DEFAULT uuid_generate_v7(),
    post_id    UUID         NOT NULL REFERENCES blogposts (id) ON DELETE CASCADE,
    parent_id  UUID REFERENCES comments (id) ON DELETE CASCADE,
    depth      INTEGER      NOT NULL DEFAULT 0,
    user_id    VARCHAR(255) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    content    TEXT         NOT NULL,
    created_at TIMESTAMPTZ           DEFAULT NOW()
);

CREATE TABLE reactions
(
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id       VARCHAR(255)     NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target_id     UUID             NOT NULL,
    target_type   target_type_enum NOT NULL,
    reaction_type reaction_type    NOT NULL,
    created_at    TIMESTAMPTZ      DEFAULT NOW(),
    UNIQUE (user_id, target_id, target_type)
);

CREATE TABLE conversations
(
    id              UUID PRIMARY KEY     DEFAULT uuid_generate_v7(),
    name            VARCHAR(100), -- Group name limit
    is_group        BOOLEAN     NOT NULL DEFAULT FALSE,
    last_message_id UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE messages
(
    id              UUID PRIMARY KEY      DEFAULT uuid_generate_v7(),
    conversation_id UUID         NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sender_id       VARCHAR(255) NOT NULL REFERENCES users (id),
    content         TEXT         NOT NULL,
    sent_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

ALTER TABLE conversations
    ADD CONSTRAINT fk_last_message
        FOREIGN KEY (last_message_id) REFERENCES messages (id) ON DELETE SET NULL;

CREATE TABLE conversation_participants
(
    conversation_id      UUID         NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id              VARCHAR(255) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    joined_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_read_message_id UUID         REFERENCES messages (id) ON DELETE SET NULL,
    PRIMARY KEY (conversation_id, user_id)
);

-- =============================================================================
-- INDEXES & TRIGGERS
-- =============================================================================

-- GIN index for ultra-fast tag filtering
CREATE INDEX idx_blogposts_tags ON blogposts USING GIN (tags);

-- Standard performance indexes
CREATE INDEX idx_blogposts_category ON blogposts (category_id);
CREATE INDEX idx_blogposts_uploader ON blogposts (uploader_id);
CREATE INDEX idx_comments_post_parent ON comments (post_id, parent_id);
CREATE INDEX idx_messages_conv_id ON messages (conversation_id, id DESC);
CREATE INDEX idx_participants_user ON conversation_participants (user_id);

-- Conversation update trigger
CREATE OR REPLACE FUNCTION func_update_conv_timestamp()
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
EXECUTE FUNCTION func_update_conv_timestamp();