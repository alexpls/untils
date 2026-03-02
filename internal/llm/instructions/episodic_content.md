## Schema and fields

Prefer using these fields if they apply:

Number: the number of the episode
Title: the title of the episode
Release date: the release date of the episode
Link: a link to the episode

## Headlines and subtitles

Headline: {{Title}}
Subtitle: Episode: {{Number}} • Release date: {{Release date}}

## Preferred sources in order of priority

For TV shows:

1. themoviedb.org
2. wikipedia.org

For podcasts:

1. The podcast website itself
2. youtube.com (if the podcast is available there)

## Examples

### Subject: Latest episode of Last Week Tonight with John Oliver

#### Fields

Number: S12E30
Release date: 2025-11-16
Title: Public Media
Link: https://www.themoviedb.org/tv/60694-last-week-tonight-with-john-oliver/season/12/episode/30

#### Result

Headline: {{Title}}
Subtitle: Episode: {{Number}} • Release date: {{Release date}}

#### Explanation

Last Week Tonight with John Oliver has clear season and episode numbers and those
are therefore included in the fields. We link to themoviedb.org directly to the episdode
as we prefer more open platforms like TMDB compared to IMDB.

### Subject: Latest episode of the WAN Show podcast

#### Fields

Release date: 2026-02-27
Title: The Linux Challenge Is Going…
Link: https://www.youtube.com/watch?v=7UGVk9ST8xw

#### Result

Headline: {{Title}}
Subtitle: Release date: {{Release date}}

#### Explanation

The WAN show does not have clear episode numbers listed, so we omit them from the fields. We link
to YouTube since that's the standard public place episodes from this podcast are uploaded and it will be
easy for the user to click on the link and listen right away.
