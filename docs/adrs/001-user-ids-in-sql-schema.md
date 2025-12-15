# User IDs in SQL schema

When creating a new database table, only add the `user_id` to it if the table represents a root entity. Skip it when it's a child entity, for example:

- `monitors` table - root entity - includes `user_id`
- `monitor_checks` table - child entity - excludes `user_id`

The intention of this is to:

1. Represent user ownership through normalised data
1. Reduce boilerplate of needing to specify `user_id` everywhere
1. Reduce size of indexes

The recognised downsides are:

1. Less defense in depth when `user_id` doesn't have to be explicitly specified in a query
1. Harder to select all of a user's entities when they're not the root
