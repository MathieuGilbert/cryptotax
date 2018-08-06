--
-- PostgreSQL database dump
--

-- Dumped from database version 9.6.3
-- Dumped by pg_dump version 9.6.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
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


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: files; Type: TABLE; Schema: public; Owner: cryptotax
--

CREATE TABLE files (
    id integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    name text NOT NULL,
    source text NOT NULL,
    bytes bytea NOT NULL,
    user_id integer NOT NULL
);


ALTER TABLE files OWNER TO cryptotax;

--
-- Name: files_id_seq; Type: SEQUENCE; Schema: public; Owner: cryptotax
--

CREATE SEQUENCE files_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE files_id_seq OWNER TO cryptotax;

--
-- Name: files_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: cryptotax
--

ALTER SEQUENCE files_id_seq OWNED BY files.id;


--
-- Name: migrations; Type: TABLE; Schema: public; Owner: cryptotax
--

CREATE TABLE migrations (
    id character varying(255) NOT NULL
);


ALTER TABLE migrations OWNER TO cryptotax;

--
-- Name: reports; Type: TABLE; Schema: public; Owner: cryptotax
--

CREATE TABLE reports (
    id integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    currency text NOT NULL
);


ALTER TABLE reports OWNER TO cryptotax;

--
-- Name: reports_id_seq; Type: SEQUENCE; Schema: public; Owner: cryptotax
--

CREATE SEQUENCE reports_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE reports_id_seq OWNER TO cryptotax;

--
-- Name: reports_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: cryptotax
--

ALTER SEQUENCE reports_id_seq OWNED BY reports.id;


--
-- Name: sessions; Type: TABLE; Schema: public; Owner: cryptotax
--

CREATE TABLE sessions (
    id integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    session_id text NOT NULL,
    csrf_token text NOT NULL,
    valid boolean NOT NULL,
    expires timestamp with time zone NOT NULL,
    user_id integer
);


ALTER TABLE sessions OWNER TO cryptotax;

--
-- Name: sessions_id_seq; Type: SEQUENCE; Schema: public; Owner: cryptotax
--

CREATE SEQUENCE sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE sessions_id_seq OWNER TO cryptotax;

--
-- Name: sessions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: cryptotax
--

ALTER SEQUENCE sessions_id_seq OWNED BY sessions.id;


--
-- Name: trades; Type: TABLE; Schema: public; Owner: cryptotax
--

CREATE TABLE trades (
    id integer NOT NULL,
    created_at timestamp with time zone NOT NULL,
    date timestamp with time zone NOT NULL,
    action text NOT NULL,
    base_currency text NOT NULL,
    file_id integer,
    currency text NOT NULL,
    amount numeric NOT NULL,
    base_amount numeric NOT NULL,
    fee_amount numeric NOT NULL,
    fee_currency text NOT NULL,
    user_id integer NOT NULL
);


ALTER TABLE trades OWNER TO cryptotax;

--
-- Name: trades_id_seq; Type: SEQUENCE; Schema: public; Owner: cryptotax
--

CREATE SEQUENCE trades_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE trades_id_seq OWNER TO cryptotax;

--
-- Name: trades_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: cryptotax
--

ALTER SEQUENCE trades_id_seq OWNED BY trades.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: cryptotax
--

CREATE TABLE users (
    id integer NOT NULL,
    email text NOT NULL,
    password text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    confirm_token text,
    confirmed boolean
);


ALTER TABLE users OWNER TO cryptotax;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: cryptotax
--

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE users_id_seq OWNER TO cryptotax;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: cryptotax
--

ALTER SEQUENCE users_id_seq OWNED BY users.id;


--
-- Name: files id; Type: DEFAULT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY files ALTER COLUMN id SET DEFAULT nextval('files_id_seq'::regclass);


--
-- Name: reports id; Type: DEFAULT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY reports ALTER COLUMN id SET DEFAULT nextval('reports_id_seq'::regclass);


--
-- Name: sessions id; Type: DEFAULT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY sessions ALTER COLUMN id SET DEFAULT nextval('sessions_id_seq'::regclass);


--
-- Name: trades id; Type: DEFAULT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY trades ALTER COLUMN id SET DEFAULT nextval('trades_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);


--
-- Name: files files_pkey; Type: CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY files
    ADD CONSTRAINT files_pkey PRIMARY KEY (id);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (id);


--
-- Name: reports reports_pkey; Type: CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY reports
    ADD CONSTRAINT reports_pkey PRIMARY KEY (id);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: trades trades_pkey; Type: CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY trades
    ADD CONSTRAINT trades_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_file_bytes_user_id; Type: INDEX; Schema: public; Owner: cryptotax
--

CREATE UNIQUE INDEX idx_file_bytes_user_id ON files USING btree (digest(bytes, 'sha1'::text), user_id);


--
-- Name: idx_user_email; Type: INDEX; Schema: public; Owner: cryptotax
--

CREATE UNIQUE INDEX idx_user_email ON users USING btree (email);


--
-- Name: files files_user_id_users_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY files
    ADD CONSTRAINT files_user_id_users_id_foreign FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT;


--
-- Name: trades trades_file_id_files_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY trades
    ADD CONSTRAINT trades_file_id_files_id_foreign FOREIGN KEY (file_id) REFERENCES files(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: trades trades_user_id_users_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: cryptotax
--

ALTER TABLE ONLY trades
    ADD CONSTRAINT trades_user_id_users_id_foreign FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--

