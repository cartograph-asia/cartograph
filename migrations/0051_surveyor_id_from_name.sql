-- New uploads are made by a signed-in Surveyor; the upload handler writes
-- the Surveyor's name into the prototype's text columns (photos.surveyor,
-- plots.first_surveyor). These triggers fill the id columns from that name
-- when the insert does not set them, so the photo and plot are attached to
-- the account without the insert itself having to change. Names are unique
-- regardless of case and never change, so the lookup is unambiguous.

create function photos_fill_surveyor_id() returns trigger
language plpgsql as $$
begin
  if new.surveyor_id is null then
    select id into new.surveyor_id from surveyors where lower(name) = lower(new.surveyor);
  end if;
  return new;
end
$$;

create trigger photos_fill_surveyor_id
before insert on photos
for each row execute function photos_fill_surveyor_id();

create function plots_fill_first_surveyor_id() returns trigger
language plpgsql as $$
begin
  if new.first_surveyor_id is null and new.first_surveyor is not null then
    select id into new.first_surveyor_id from surveyors where lower(name) = lower(new.first_surveyor);
  end if;
  return new;
end
$$;

create trigger plots_fill_first_surveyor_id
before insert on plots
for each row execute function plots_fill_first_surveyor_id();
