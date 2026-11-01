-- Magic-link sign-in and sessions. Only SHA-256 hashes of the tokens are
-- stored; the raw token exists in the e-mail link and the cookie only.

create table login_tokens (
  token_hash bytea primary key check (length(token_hash) = 32),
  email      text not null,         -- lower case; the account may not exist yet
  next_path  text,                  -- local path to return to after sign-in
  created_at timestamptz not null default now(),
  expires_at timestamptz not null,
  used_at    timestamptz            -- set when redeemed; a used token never works again
);

create index login_tokens_email_idx on login_tokens(email, created_at desc);
create index login_tokens_expires_idx on login_tokens(expires_at);

-- A session starts when a magic link is redeemed. surveyor_id is null until
-- a first-time visitor has chosen a Surveyor name.
create table sessions (
  token_hash   bytea primary key check (length(token_hash) = 32),
  email        text not null,
  surveyor_id  uuid references surveyors(id) on delete cascade,
  created_at   timestamptz not null default now(),
  expires_at   timestamptz not null,
  last_seen_at timestamptz not null default now()
);

create index sessions_surveyor_idx on sessions(surveyor_id);
create index sessions_expires_idx on sessions(expires_at);
