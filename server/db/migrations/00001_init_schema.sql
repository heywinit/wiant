-- +goose Up
CREATE TABLE hosts (
    host_id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE agents (
    agent_id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    host_id BIGINT NOT NULL REFERENCES hosts(host_id)
);

-- +goose Down
DROP TABLE agents;
DROP TABLE hosts;
