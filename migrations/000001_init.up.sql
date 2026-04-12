CREATE TABLE users (
    id            UUID PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE wishlists (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    event_date   TIMESTAMPTZ NOT NULL,
    public_token UUID NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE items (
    id          UUID PRIMARY KEY,
    wishlist_id UUID NOT NULL REFERENCES wishlists(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    url         TEXT NOT NULL DEFAULT '',
    priority    SMALLINT NOT NULL CHECK (priority BETWEEN 1 AND 5),
    is_reserved BOOLEAN NOT NULL DEFAULT FALSE,
    reserved_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wishlists_user_id ON wishlists(user_id);
CREATE INDEX idx_wishlists_public_token ON wishlists(public_token);
CREATE INDEX idx_items_wishlist_id ON items(wishlist_id);