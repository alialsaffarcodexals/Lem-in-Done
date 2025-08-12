-- Users & sessions
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,              -- UUID
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Content
CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Categories
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS post_categories (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, category_id),
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
);

-- Likes (posts & comments)
CREATE TABLE IF NOT EXISTS likes (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    target_type TEXT NOT NULL CHECK (target_type IN ('post','comment')),
    target_id INTEGER NOT NULL,
    value INTEGER NOT NULL CHECK (value IN (-1,1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, target_type, target_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Seed a few categories (idempotent)
INSERT INTO categories (name)
    SELECT 'General' WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name='General');
INSERT INTO categories (name)
    SELECT 'Help' WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name='Help');
INSERT INTO categories (name)
    SELECT 'Off-topic' WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name='Off-topic');
