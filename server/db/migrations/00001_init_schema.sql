-- +goose Up
CREATE TABLE hosts (
    host_id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE agents (
    agent_id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    host_id BIGSERIAL references hosts(agent_id)
);

-- +goose Down
DROP TABLE hosts;
DROP TABLE agents;
