-- Finer place types for plot names: suburb, quarter and neighbourhood join
-- city, town and village. A plot is named after the nearest place of the
-- finest type within 2 km (neighbourhood, then quarter, then suburb), else
-- the nearest village or town within 5 km, else a city within 5 km, else
-- its province (internal/db, nearestPlaceSQL).
alter table places drop constraint places_kind_check;
alter table places add constraint places_kind_check
  check (kind in ('city', 'town', 'village', 'suburb', 'quarter', 'neighbourhood'));
