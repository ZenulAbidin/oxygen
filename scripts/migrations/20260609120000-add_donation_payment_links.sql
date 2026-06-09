-- +migrate Up
alter table payment_links
    add column if not exists type varchar(16) default 'payment' not null;

alter table payment_links
    alter column price drop not null;

alter table payment_links
    alter column decimals set default 2;

update payment_links
set type = 'payment'
where type = '';

-- +migrate Down
update payment_links
set price = 0
where price is null;

alter table payment_links
    alter column price set not null;

alter table payment_links
    alter column decimals drop default;

alter table payment_links
    drop column if exists type;
