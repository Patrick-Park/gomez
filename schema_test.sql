--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = public, pg_catalog;

--
-- Name: message_ids; Type: SEQUENCE SET; Schema: public; Owner: Gabriel
--

SELECT pg_catalog.setval('message_ids', 1, false);


--
-- Data for Name: messages; Type: TABLE DATA; Schema: public; Owner: Gabriel
--

COPY messages (id, "from", rcpt, raw) FROM stdin;
\.


--
-- Data for Name: queue; Type: TABLE DATA; Schema: public; Owner: Gabriel
--

COPY queue (message_id, rcpt, date_added, attempts) FROM stdin;
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: Gabriel
--

COPY users (id, name, address) FROM stdin;
1	james john	james@john.com
2	adam carry	adam@carry.com
11	jim quick	jim@quick.com
\.


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: Gabriel
--

SELECT pg_catalog.setval('users_id_seq', 11, true);


--
-- PostgreSQL database dump complete
--

