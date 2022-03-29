-- The idea for the project is pretty simple. It is a URL shortener like bitly. A user provedies a URL to shorten,
-- the app returns a short URL. When the short URL is used the server redirects the user to full URL and writes data 
-- to a database. Right now it only writes a number of redirects, but there can be more data that can be gathered from redirects.

--tables

--table:urlbase
--table where we store all the URLs that we generated and short URLs that relate to them. Values Updated.
BEGIN;

-- create database bitmedb;

-- create user bituser password 'bit';

-- grant all privileges on database bitmedb to bituser;

-- \c bitmedb bituser;

CREATE schema bitme;


CREATE TABLE bitme.urlbase (
    short_url varchar(10) PRIMARY KEY,
    full_url text NOT NULL  
);

--table:urldata
--table that stores primary data for short URLs - number of time used and how active it is. Values updated.
CREATE TABLE bitme.urldata(
    short_url varchar(10) PRIMARY KEY,    
    last_used timestamp NOT NULL,
    total_num_of_uses int NOT NULL,    
    FOREIGN KEY (short_url) REFERENCES bitme.urlbase(short_url),
    constraint total_num_of_uses_check check (total_num_of_uses >= 0)
);

--table:adminurl
--table that stores urls for getting info by admins
CREATE TABLE bitme.adminurl (
    admin_url varchar(10) PRIMARY KEY,
    short_url varchar(10) NOT NULL,  
    FOREIGN KEY (short_url) REFERENCES bitme.urlbase(short_url)
);

--table urlusedata
--table that stores uses of short url from certain ip adresses
CREATE TABLE bitme.urlusedata(
    ip text NOT NULL ,
    short_url varchar(10) NOT NULL,   
    last_used timestamp NOT NULL,
    ip_num_of_uses int NOT NULL,
    FOREIGN KEY (short_url) REFERENCES bitme.urlbase(short_url),
    constraint ip_num_of_uses_check check (ip_num_of_uses >= 0)
);

--INDEX
--as the app is an URL sortener a must have inde is for bitme.urlbase that has data for redirects
--so it will has much more reads than writes. 
--Another logical index is for bitme.adminurl for quick access to your admin page.
--However they are already PRIMARY KEYS so no need to create seperate indexes for them.
--An index that can be usefull is an index for IP table in bitme.urlusedata, because we need to quickly get all ips for admin page
--may be useless since there is short_url as primary_key.

create index concurrently ip_id on bitme.urlusedata(ip);

COMMIT;



