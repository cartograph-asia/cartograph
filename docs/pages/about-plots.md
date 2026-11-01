# About the plots

Cartograph divides the map into **plots**: hexagons of about 0.74 km², roughly 920 m from corner to corner and 800 m from edge to edge. A plot is small enough that a village has several and a city has hundreds, and large enough that each one holds a real piece of landscape – a stretch of coast, a few streets, a hillside. You can walk across one in ten minutes.

## Why hexagons

Squares look tidy, but on a map they cheat: a square's diagonal neighbour is further away than its side neighbour, so "next to" means two different things. Every hexagon has six neighbours at exactly the same distance. That makes neighbouring plots, routes between them and the look of a filling map simpler and fairer. Hexagons also tile a sphere with far less distortion than squares do.

## Where the grid comes from

The plots are cells of [H3](https://h3geo.org/), an open hierarchical grid system developed at Uber and used widely in mapping and analysis. H3 projects the Earth onto an icosahedron and tiles it with hexagons at sixteen resolutions. Cartograph uses resolution 8 for plots.

A few numbers about resolution 8:

| | |
|---|---|
| Cells worldwide | 691,776,122 |
| Average area | 0.737 km² |
| Average edge length | 461 m |
| Cells on land in Cartograph's region | about 30 million |

Every cell has a permanent, globally unique index such as `88658bd337fffff`. You will see it in the details of a plot page; it is what links Cartograph's data to any other H3-based dataset in the world.

## The twelve pentagons

A sphere cannot be tiled with hexagons alone. At every resolution H3 has exactly twelve pentagons, sitting at the corners of the icosahedron. The designers of H3 oriented the icosahedron so that all twelve corners fall in the ocean. There is not a single pentagonal plot anywhere in Cartograph's region.

## Plot names

Nobody should have to say "eight-eight-six-five-eight-b-d". Every plot has a readable name made of the nearest place and a number within its province, such as **Pratumnak Hill · Chonburi 4812**. Numbers are assigned along a space-filling curve, so neighbouring plots have neighbouring numbers: 4810 to 4815 is one stretch of coast.

## Zooming out

H3 is hierarchical: seven resolution-8 plots sit inside one resolution-7 cell, and so on upward. When you zoom out, Cartograph shows coverage per larger cell – how many of its plots have been surveyed – which is how the map of a whole province or country turns from blank to green over the years.

## Surveying a plot

A plot is **surveyed** once it has at least one photo taken inside it. The first photo **claims** the plot, and the Surveyor who took it stays its first surveyor for good. Further photos add to the plot: other seasons, other corners, other years. There is no limit; the goal is every plot, and then every plot in every season.
