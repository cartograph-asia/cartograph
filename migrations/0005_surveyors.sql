-- Surveyors: one account per member. The name is chosen once, is unique
-- regardless of case and is the URL (/surveyor/{name}). Placeholder accounts
-- for prototype photos have no e-mail until Adrian sets one in SQL.

create table surveyors (
  id             uuid primary key,
  name           text not null check (name ~ '^[A-Za-z0-9-]{3,30}$'),
  email          text,                    -- normalised to lower case by the backend; null for unclaimed placeholders
  placeholder    boolean not null default false, -- created from a prototype free-text name, not by sign-up
  bio            text not null default '' check (char_length(bio) <= 500),
  home_region_id int references regions(id) on delete set null,
  joined_at      timestamptz not null default now()
);

create unique index surveyors_name_lower_idx on surveyors(lower(name));
create unique index surveyors_email_lower_idx on surveyors(lower(email));
create index surveyors_joined_idx on surveyors(joined_at desc);
