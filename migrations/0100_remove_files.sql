-- Removing the image files of a hidden photo (docs/ADMIN_BRIEF.md, decision
-- 2). The row, its hidden state and the audit trail stay; only the objects
-- in the bucket go. A photo whose files are removed cannot be unhidden.
alter table photos
  add column files_removed_at timestamptz,
  add check (files_removed_at is null or hidden_at is not null);

-- The view lists photos' columns as they were when it was created; add the
-- new one.
create or replace view visible_photos as select * from photos where hidden_at is null;

alter table admin_actions drop constraint admin_actions_action_check;
alter table admin_actions add constraint admin_actions_action_check check (action in (
  'photo.hide', 'photo.unhide', 'photo.remove_files',
  'featured.create', 'featured.end', 'featured.delete',
  'surveyor.block', 'surveyor.unblock', 'surveyor.end_sessions', 'surveyor.clear_bio', 'surveyor.clear_display_name',
  'place_override.create', 'place_override.update',
  'invite.create', 'invite.resend', 'invite.revoke'));
