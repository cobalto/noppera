CREATE TABLE boards (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(10) NOT NULL UNIQUE,
    description TEXT,
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    board_id INTEGER NOT NULL REFERENCES boards(id),
    thread_id INTEGER REFERENCES posts(id),
    user_id INTEGER,
    title VARCHAR(200),
    content TEXT NOT NULL,
    image_url TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,
    last_bumped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    archived_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE flags (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL REFERENCES posts(id),
    user_id INTEGER,
    reason TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_posts_board_id ON posts(board_id);
CREATE INDEX idx_posts_thread_id ON posts(thread_id);
CREATE INDEX idx_posts_last_bumped_at ON posts(last_bumped_at);
CREATE INDEX idx_posts_archived_at ON posts(archived_at);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_posts_content_fts ON posts USING GIN (to_tsvector('english', content));

INSERT INTO boards (name, slug, description, settings) VALUES
('General', 'g', 'General discussion', '{"max_threads": 100, "max_replies": 500, "max_image_size": 5242880}'),
('Tech', 't', 'Technology and gadgets', '{"max_threads": 50, "max_replies": 300, "max_image_size": 3145728}');

INSERT INTO users (username, password, is_admin)
VALUES ('admin', '$2a$10$3zH0y3zH0y3zH0y3zH0y3u3zH0y3zH0y3zH0y3zH0y3zH0y3zH0y', TRUE)
ON CONFLICT (username) DO NOTHING;