--
-- PostgreSQL database dump
--

-- Dumped from database version 10.4
-- Dumped by pg_dump version 10.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: discordusers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.discordusers (
    uid integer NOT NULL,
    userid character varying(100) NOT NULL,
    username character varying(100) NOT NULL,
    discriminator character varying(500) NOT NULL,
    xp integer,
    nextxpgaintime timestamp without time zone,
    xpexcluded boolean,
    reputation integer,
    cangivereptime timestamp without time zone
);


ALTER TABLE public.discordusers OWNER TO postgres;

--
-- Name: discordusers_uid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.discordusers_uid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.discordusers_uid_seq OWNER TO postgres;

--
-- Name: discordusers_uid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.discordusers_uid_seq OWNED BY public.discordusers.uid;


--
-- Name: userroles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.userroles (
    uid integer NOT NULL,
    guildid character varying(100) NOT NULL,
    userid character varying(100) NOT NULL,
    roleid character varying(500) NOT NULL
);


ALTER TABLE public.userroles OWNER TO postgres;

--
-- Name: userroles_uid_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.userroles_uid_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.userroles_uid_seq OWNER TO postgres;

--
-- Name: userroles_uid_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.userroles_uid_seq OWNED BY public.userroles.uid;


--
-- Name: discordusers uid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.discordusers ALTER COLUMN uid SET DEFAULT nextval('public.discordusers_uid_seq'::regclass);


--
-- Name: userroles uid; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userroles ALTER COLUMN uid SET DEFAULT nextval('public.userroles_uid_seq'::regclass);


--
-- Data for Name: discordusers; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.discordusers (uid, userid, username, discriminator, xp, nextxpgaintime, xpexcluded, reputation, cangivereptime) FROM stdin;
\.


--
-- Data for Name: userroles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.userroles (uid, guildid, userid, roleid) FROM stdin;
\.


--
-- Name: discordusers_uid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.discordusers_uid_seq', 1, false);


--
-- Name: userroles_uid_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.userroles_uid_seq', 1, false);


--
-- Name: discordusers discorduser_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.discordusers
    ADD CONSTRAINT discorduser_pkey PRIMARY KEY (uid);


--
-- Name: userroles selfrole_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userroles
    ADD CONSTRAINT selfrole_pkey PRIMARY KEY (uid);


--
-- PostgreSQL database dump complete
--

