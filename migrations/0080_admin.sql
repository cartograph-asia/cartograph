-- The admin area of docs/ADMIN_BRIEF.md: hidden photos, blocked Surveyors,
-- invitations and the audit trail of every admin action.

-- A hidden photo is gone from every public read. It is never deleted; the
-- plot's claim and counts are recomputed from the visible photos when a
-- photo is hidden or unhidden (internal/db/admin.go).
alter table photos
  add column hidden_at     timestamptz,
  add column hidden_reason text check (char_length(hidden_reason) <= 200),
  add check ((hidden_at is null) = (hidden_reason is null));

create index photos_hidden_idx on photos(hidden_at desc) where hidden_at is not null;

-- Every public query reads this view instead of photos. "select *" is
-- expanded when the view is created: a migration that adds a column to
-- photos recreates the view.
create view visible_photos as select * from photos where hidden_at is null;

-- A blocked Surveyor has no sessions and gets none; their profile and
-- photos stay public unless hidden.
alter table surveyors
  add column blocked_at     timestamptz,
  add column blocked_reason text check (char_length(blocked_reason) <= 200),
  add check ((blocked_at is null) = (blocked_reason is null));

-- While SIGNUP=invite a new account needs an open invitation (not revoked,
-- not accepted) for its address. One row per address; inviting again after
-- a revoke reopens the row.
create table invitations (
  email       text primary key check (email = lower(email)),
  invited_by  uuid not null references surveyors(id),
  note        text not null default '' check (char_length(note) <= 300),
  invited_at  timestamptz not null,
  accepted_at timestamptz,
  accepted_by uuid references surveyors(id),
  revoked_at  timestamptz,
  check ((accepted_at is null) = (accepted_by is null))
);

create index invitations_invited_idx on invitations(invited_at desc);

-- Addresses that asked to sign in without an account or an invitation got
-- the invitation-only mail at most once per day. Only a SHA-256 of the
-- normalised address is kept.
create table signup_refusals (
  email_hash bytea primary key check (length(email_hash) = 32),
  sent_at    timestamptz not null
);

-- The audit trail. Written in the same transaction as the change it
-- records; rows are never updated or deleted.
create table admin_actions (
  id       bigint generated always as identity primary key,
  at       timestamptz not null default now(),
  admin_id uuid not null references surveyors(id),
  action   text not null check (action in (
             'photo.hide', 'photo.unhide',
             'featured.create', 'featured.end', 'featured.delete',
             'surveyor.block', 'surveyor.unblock', 'surveyor.end_sessions', 'surveyor.clear_bio',
             'place_override.create', 'place_override.update',
             'invite.create', 'invite.resend', 'invite.revoke')),
  target   text not null,
  reason   text not null default '',
  before   jsonb,
  after    jsonb
);

create index admin_actions_at_idx on admin_actions(at desc, id desc);
create index admin_actions_target_idx on admin_actions(target, at desc);

create function admin_actions_append_only() returns trigger language plpgsql as $$
begin
  raise exception 'admin_actions is append-only';
end $$;

create trigger admin_actions_append_only before update or delete on admin_actions
  for each statement execute function admin_actions_append_only();
