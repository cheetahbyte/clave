-- +goose Up
-- +goose StatementBegin

-- Deduplicate before adding unique constraint (keeps earliest activation per device)
delete from activations a
using activations b
where a.id > b.id
  and a.license_id = b.license_id
  and a.device_id = b.device_id;

create unique index if not exists activations_license_device_uq
    on activations (license_id, device_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop index if exists activations_license_device_uq;

-- +goose StatementEnd
