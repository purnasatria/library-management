#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE library_auth;
    CREATE DATABASE library_author;
    CREATE DATABASE library_category;
    CREATE DATABASE library_book;
EOSQL
