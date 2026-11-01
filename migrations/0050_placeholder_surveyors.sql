-- Attach the prototype's photos to accounts. Every distinct free-text
-- photos.surveyor without a surveyor_id becomes a placeholder Surveyor
-- (no e-mail; Adrian claims one by setting its e-mail in SQL, and its first
-- sign-in clears the placeholder flag).
--
-- The name is slugified to the Surveyor name rules (3-30 of [A-Za-z0-9-]):
-- common Latin accents are folded, every other run of characters becomes
-- one hyphen, hyphens at the ends are dropped. A name with nothing left
-- (e.g. only Thai script) becomes "surveyor"; one or two characters get a
-- "-1" suffix. On a case-insensitive collision with an existing name the
-- suffix counts up from "-2". Names are handled in order of their first
-- upload, so the earliest surveyor keeps the plain name. joined_at is that
-- first upload.

do $$
declare
  rec  record;
  base text;
  cand text;
  n    int;
  sid  uuid;
begin
  for rec in
    select surveyor, min(uploaded_at) as first_upload
    from photos
    where surveyor_id is null
    group by surveyor
    order by min(uploaded_at), surveyor
  loop
    base := translate(btrim(rec.surveyor),
      'ÀÁÂÃÄÅàáâãäåĀāĂăĄąÇçĆćČčĎďÈÉÊËèéêëĒēĖėĘęĚěÌÍÎÏìíîïĪīĮįŁłÑñŃńŇňÒÓÔÕÖØòóôõöøŌōŐőŘřŚśŠšŤťÙÚÛÜùúûüŪūŮůŰűÝýÿŹźŻżŽž',
      'AAAAAAaaaaaaAaAaAaCcCcCcDdEEEEeeeeEeEeEeEeIIIIiiiiIiIiLlNnNnNnOOOOOOooooooOoOoRrSsSsTtUUUUuuuuUuUuUuYyyZzZzZz');
    base := replace(replace(replace(base, 'ß', 'ss'), 'Æ', 'AE'), 'æ', 'ae');
    base := regexp_replace(base, '[^A-Za-z0-9]+', '-', 'g');
    base := btrim(base, '-');
    base := rtrim(left(base, 30), '-');
    if base = '' then
      base := 'surveyor';
    end if;

    n := case when length(base) < 3 then 1 else 0 end;
    loop
      cand := case when n = 0 then base
                   else rtrim(left(base, 29 - length(n::text)), '-') || '-' || n end;
      exit when not exists (select 1 from surveyors where lower(name) = lower(cand));
      n := case when n = 0 then 2 else n + 1 end;
    end loop;

    sid := gen_random_uuid();
    insert into surveyors (id, name, placeholder, joined_at)
    values (sid, cand, true, rec.first_upload);

    update photos set surveyor_id = sid
    where surveyor = rec.surveyor and surveyor_id is null;
  end loop;
end
$$;

update plots pl
set first_surveyor_id = ph.surveyor_id
from photos ph
where ph.id = pl.first_photo and pl.first_surveyor_id is null;
