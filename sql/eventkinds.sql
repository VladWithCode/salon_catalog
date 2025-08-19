--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.5 (Debian 17.5-1.pgdg130+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: event_kinds; Type: TABLE DATA; Schema: public; Owner: vladwb
--

COPY public.event_kinds (id, name, description, created_at, updated_at) FROM stdin;
01989668-2ba8-7042-bd28-c7b0c9f00544	Boda		2025-08-10 17:54:26.088201	2025-08-10 17:54:26.088201
01989668-7440-710f-bcfa-a87ee8eeadbf	Quinceañera		2025-08-10 17:54:44.672137	2025-08-10 17:54:44.672137
01989668-9c18-7811-8b5c-709d8ff3d937	Bautizo		2025-08-10 17:54:54.872598	2025-08-10 17:54:54.872598
01989668-be90-7758-a84c-d4be32abbfe3	Fiesta Privada		2025-08-10 17:55:03.696537	2025-08-10 17:55:03.696537
01989669-aec2-7858-9f2e-8275c7154dac	Graduación		2025-08-10 17:56:05.186593	2025-08-10 17:56:05.186593
\.


--
-- PostgreSQL database dump complete
--

