--
-- PostgreSQL database dump
--

-- Dumped from database version 9.3.4
-- Dumped by pg_dump version 9.3.4
-- Started on 2014-10-20 19:00:47 BST

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- TOC entry 2226 (class 1262 OID 16385)
-- Name: gomez; Type: DATABASE; Schema: -; Owner: -
--

DROP DATABASE gomez_test
CREATE DATABASE gomez_test WITH TEMPLATE = template0 ENCODING = 'UTF8' LC_COLLATE = 'en_US.UTF-8' LC_CTYPE = 'en_US.UTF-8';


\connect gomez_test

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- TOC entry 176 (class 3079 OID 12018)
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- TOC entry 2228 (class 0 OID 0)
-- Dependencies: 176
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- TOC entry 175 (class 1259 OID 18404)
-- Name: mailbox; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE mailbox (
    user_id bigint NOT NULL,
    message_id bigint NOT NULL
);


--
-- TOC entry 174 (class 1259 OID 16419)
-- Name: message_ids; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE message_ids
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 172 (class 1259 OID 16403)
-- Name: messages; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE messages (
    id bigint NOT NULL,
    "from" character varying(255) NOT NULL,
    rcpt character varying NOT NULL,
    raw text NOT NULL
);


--
-- TOC entry 173 (class 1259 OID 16411)
-- Name: queue; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE queue (
    message_id bigint NOT NULL,
    rcpt character varying NOT NULL,
    date_added timestamp without time zone NOT NULL,
    attempts integer
);


--
-- TOC entry 171 (class 1259 OID 16388)
-- Name: users; Type: TABLE; Schema: public; Owner: -; Tablespace: 
--

CREATE TABLE users (
    id bigint NOT NULL,
    name character varying(255),
    address character varying(255)
);


--
-- TOC entry 170 (class 1259 OID 16386)
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 2229 (class 0 OID 0)
-- Dependencies: 170
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE users_id_seq OWNED BY users.id;


--
-- TOC entry 2106 (class 2604 OID 16423)
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);


--
-- TOC entry 2108 (class 2606 OID 16398)
-- Name: address; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT address UNIQUE (address);


--
-- TOC entry 2114 (class 2606 OID 18408)
-- Name: inbox; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY mailbox
    ADD CONSTRAINT inbox UNIQUE (message_id, user_id);


--
-- TOC entry 2112 (class 2606 OID 16410)
-- Name: messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY messages
    ADD CONSTRAINT messages_pkey PRIMARY KEY (id);


--
-- TOC entry 2110 (class 2606 OID 16396)
-- Name: users_pkey; Type: CONSTRAINT; Schema: public; Owner: -; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


-- Completed on 2014-10-20 19:00:47 BST

--
-- PostgreSQL database dump complete
--

