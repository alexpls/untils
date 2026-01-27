ALTER TABLE public.monitors
ADD COLUMN check_schedule text NOT NULL DEFAULT '0 */6 * * *';
