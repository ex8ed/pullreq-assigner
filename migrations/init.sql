CREATE TABLE IF NOT EXISTS teams (
    name        VARCHAR(255) PRIMARY KEY
);


CREATE TABLE IF NOT EXISTS users (
    id          VARCHAR(255) PRIMARY KEY,
    username    VARCHAR(255) NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    team_name   VARCHAR(255) NOT NULL,
    CONSTRAINT fk_team FOREIGN KEY (team_name) REFERENCES teams(name) ON DELETE RESTRICT
);


CREATE TABLE IF NOT EXISTS pull_requests (
    id          VARCHAR(255) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    author_id   VARCHAR(255) NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'OPEN',
    created_at  TIMESTAMP    DEFAULT NOW(),
    merged_at   TIMESTAMP

    CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT
);


CREATE TABLE IF NOT EXISTS pr_reviewers (
    pull_request_id VARCHAR(255) NOT NULL,
    user_id         VARCHAR(255) NOT NULL,
    
    PRIMARY KEY (pull_request_id, user_id)

    CONSTRAINT fk_pr FOREIGN KEY (pull_request_id) REFERENCES pull_requests(id) ON DELETE CASCADE,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);
