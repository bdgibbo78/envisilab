BEGIN TRANSACTION;

CREATE SCHEMA v1;

-- Entities map the relationship between a persistent client UUID and
-- a per-session token UUID.
CREATE TABLE v1.entity
(
    client_uuid UUID NOT NULL,
    token_uuid UUID NOT NULL PRIMARY KEY,
    type TEXT NOT NULL,
    created TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

CREATE TABLE v1.location_data
(
    token_uuid UUID NOT NULL REFERENCES v1.entity ON DELETE CASCADE,
    lat FLOAT(8) NOT NULL,
    lng FLOAT(8) NOT NULL,
    alt REAL NOT NULL,
    timestamp TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

-- Permissions
GRANT USAGE ON SCHEMA v1 TO data_producer;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA v1 TO data_producer;
GRANT SELECT, UPDATE, INSERT ON ALL TABLES IN SCHEMA v1 TO data_producer;

END TRANSACTION;
