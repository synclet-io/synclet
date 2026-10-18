-- +goose Up
-- +goose StatementBegin
DELETE FROM pipeline.repositories old
WHERE old.url = 'https://synclet-io.github.io/synclet-connector-registry/registry.json'
    AND EXISTS (
        SELECT 1 FROM pipeline.repositories cur
        WHERE cur.workspace_id = old.workspace_id
            AND cur.url = 'https://synclet-io.github.io/connectors/registry.json'
    );

UPDATE pipeline.repositories
SET url = 'https://synclet-io.github.io/connectors/registry.json', updated_at = NOW()
WHERE url = 'https://synclet-io.github.io/synclet-connector-registry/registry.json';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE pipeline.repositories
SET url = 'https://synclet-io.github.io/synclet-connector-registry/registry.json', updated_at = NOW()
WHERE url = 'https://synclet-io.github.io/connectors/registry.json';
-- +goose StatementEnd
