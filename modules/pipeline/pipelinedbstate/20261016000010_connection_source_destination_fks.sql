-- +goose Up
-- +goose StatementBegin
-- NOT VALID enforces the keys for every new or changed row and for deletes of
-- sources and destinations, without failing on connections that already point
-- at a deleted source or destination. Run VALIDATE CONSTRAINT once those rows
-- are cleaned up.
ALTER TABLE pipeline.connections
    ADD CONSTRAINT connections_source_id_fkey
    FOREIGN KEY (source_id) REFERENCES pipeline.sources(id) ON DELETE RESTRICT NOT VALID;

ALTER TABLE pipeline.connections
    ADD CONSTRAINT connections_destination_id_fkey
    FOREIGN KEY (destination_id) REFERENCES pipeline.destinations(id) ON DELETE RESTRICT NOT VALID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE pipeline.connections DROP CONSTRAINT connections_destination_id_fkey;
ALTER TABLE pipeline.connections DROP CONSTRAINT connections_source_id_fkey;
-- +goose StatementEnd
