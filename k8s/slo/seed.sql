BEGIN;

INSERT INTO users (id, email, password_hash)
VALUES (
    '7bfa2e15-8b3b-4c53-a162-d17c539db543',
    'sre-prober@wishlist.invalid',
    '!'
)
ON CONFLICT (id) DO NOTHING;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM users
        WHERE id = '7bfa2e15-8b3b-4c53-a162-d17c539db543'
          AND email = 'sre-prober@wishlist.invalid'
          AND password_hash = '!'
    ) THEN
        RAISE EXCEPTION 'Reserved prober user ID is already in use';
    END IF;
END $$;

INSERT INTO wishlists (id, user_id, title, description, event_date, public_token)
VALUES (
    'fab051b0-98d8-46fa-8a8d-6a4c7df19623',
    '7bfa2e15-8b3b-4c53-a162-d17c539db543',
    'Контрольный вишлист пробера',
    'Тестовые данные для проверки публичного чтения',
    NOW() + INTERVAL '1 year',
    gen_random_uuid()
)
ON CONFLICT (id) DO NOTHING;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM wishlists
        WHERE id = 'fab051b0-98d8-46fa-8a8d-6a4c7df19623'
          AND user_id = '7bfa2e15-8b3b-4c53-a162-d17c539db543'
    ) THEN
        RAISE EXCEPTION 'Reserved prober wishlist ID is already in use';
    END IF;
END $$;

COMMIT;

SELECT 'PROBE_URL=http://wishlist-service/api/v1/public/' || public_token
FROM wishlists WHERE id = 'fab051b0-98d8-46fa-8a8d-6a4c7df19623';

SELECT 'PROBE_EXPECTED_WISHLIST_ID=' || id
FROM wishlists WHERE id = 'fab051b0-98d8-46fa-8a8d-6a4c7df19623';
