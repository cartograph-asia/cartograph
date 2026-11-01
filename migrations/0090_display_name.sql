-- An optional display name beside the Surveyor name (docs/DISPLAY_NAME_BRIEF.md).
-- The Surveyor name stays the handle in URLs and in the copies in photos and
-- plots; the display name is read live from here wherever names are shown.
alter table surveyors
  add column display_name text check (display_name is null or char_length(display_name) between 1 and 50);

alter table admin_actions drop constraint admin_actions_action_check;
alter table admin_actions add constraint admin_actions_action_check check (action in (
  'photo.hide', 'photo.unhide',
  'featured.create', 'featured.end', 'featured.delete',
  'surveyor.block', 'surveyor.unblock', 'surveyor.end_sessions', 'surveyor.clear_bio', 'surveyor.clear_display_name',
  'place_override.create', 'place_override.update',
  'invite.create', 'invite.resend', 'invite.revoke'));
