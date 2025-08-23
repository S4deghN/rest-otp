SET @@sql_mode := REPLACE(@@sql_mode, 'NO_ZERO_DATE', '');

DROP DATABASE IF EXISTS db;
CREATE DATABASE db;
USE db;

create table Users
(
    id        varchar(20) primary key,
    state     int default 0 not NULL,
    reg_date  datetime NULL,
    otp_val   varchar(10) NULL,
    otp_exp   datetime NULL,
    otp_tries int NULL,
    otp_first datetime NULL
);
